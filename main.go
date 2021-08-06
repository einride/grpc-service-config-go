package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/compiler/protogen"
)

const docURL = "https://github.com/grpc/grpc/blob/master/doc/service_config.md"

func main() {
	var (
		flags    flag.FlagSet
		path     = flags.String("path", "", "input path of service config JSON files")
		validate = flags.Bool("validate", false, "validate service configs")
		required = flags.Bool("required", false, "require every service to have a service config")
	)
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		p := plugin{
			gen:  gen,
			path: *path,
		}
		if *validate {
			if err := p.validate(*required); err != nil {
				return err
			}
		}
		return p.run()
	})
}

type plugin struct {
	gen  *protogen.Plugin
	path string
}

func (p *plugin) run() error {
	generatedServiceConfigFiles := map[string]struct{}{}
	for _, file := range p.gen.Files {
		if !file.Generate {
			continue
		}
		for _, service := range file.Services {
			serviceConfigFile := p.resolveServiceConfigFile(service)
			if _, err := os.Stat(serviceConfigFile); err != nil {
				continue
			}
			if _, ok := generatedServiceConfigFiles[serviceConfigFile]; ok {
				continue
			}
			generatedServiceConfigFiles[serviceConfigFile] = struct{}{}
			data, err := ioutil.ReadFile(serviceConfigFile)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(data, &serviceConfig{}); err != nil {
				return fmt.Errorf("run: invalid service config file %s: %w", serviceConfigFile, err)
			}
			g := p.gen.NewGeneratedFile(
				filepath.Dir(file.GeneratedFilenamePrefix)+"/"+filepath.Base(serviceConfigFile)+".go",
				file.GoImportPath,
			)
			g.P("package ", file.GoPackageName)
			g.P()
			g.P("// ServiceConfig is the service config for all services in the package.")
			g.P("// Source: ", filepath.Base(serviceConfigFile), ".")
			g.P("const ServiceConfig = `", string(data), "`")
		}
	}
	return nil
}

func (p *plugin) resolveServiceConfigFile(service *protogen.Service) string {
	parentPackageName := string(service.Desc.ParentFile().Package().Parent().Name())
	fileName := parentPackageName + "_grpc_service_config.json"
	return filepath.Join(p.path, filepath.Dir(service.Location.SourceFile), fileName)
}

func (p *plugin) validate(required bool) error {
	addr, cleanup, err := p.startLocalServer()
	if err != nil {
		return err
	}
	defer cleanup()
	for _, file := range p.gen.Files {
		if !file.Generate {
			continue
		}
		for _, service := range file.Services {
			serviceConfigFile := p.resolveServiceConfigFile(service)
			if required {
				if _, err := os.Stat(serviceConfigFile); err != nil {
					return fmt.Errorf(
						"validate: missing service config file %s for %s (see: %s)",
						serviceConfigFile,
						service.Desc.FullName(),
						docURL,
					)
				}
			}
			data, err := ioutil.ReadFile(serviceConfigFile)
			if err != nil {
				return err
			}
			// gRPC Go validates a service config when dialing.
			conn, err := grpc.Dial(
				addr,
				grpc.WithDefaultServiceConfig(string(data)),
				grpc.WithInsecure(),
				grpc.WithBlock(),
			)
			if err != nil {
				return fmt.Errorf("validate: invalid service config %s: %w", serviceConfigFile, err)
			}
			if err := conn.Close(); err != nil {
				return err
			}
			var serviceConfigContent serviceConfig
			if err := json.Unmarshal(data, &serviceConfigContent); err != nil {
				return err
			}
			if required && !serviceConfigContent.hasService(service) {
				return fmt.Errorf(
					"validate: missing service config for %s in %s (see: %s)",
					service.Desc.FullName(),
					serviceConfigFile,
					docURL,
				)
			}
		}
	}
	return nil
}

type serviceConfig struct {
	MethodConfigs []struct {
		Names []struct {
			Service string
			Method  string
		} `json:"name"`
	} `json:"methodConfig"`
}

func (c serviceConfig) hasService(service *protogen.Service) bool {
	for _, methodConfig := range c.MethodConfigs {
		for _, name := range methodConfig.Names {
			if (name.Service == "" && name.Method == "") ||
				(name.Service == string(service.Desc.FullName()) && name.Method == "") {
				return true
			}
		}
	}
	return false
}

func (p *plugin) startLocalServer() (string, func(), error) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", nil, err
	}
	localServer := grpc.NewServer()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = localServer.Serve(lis)
	}()
	cleanup := func() {
		localServer.Stop()
		wg.Wait()
	}
	return lis.Addr().String(), cleanup, nil
}
