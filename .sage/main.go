package main

import (
	"context"
	"os"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgbuf"
	"go.einride.tech/sage/tools/sgconvco"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sggo"
	"go.einride.tech/sage/tools/sggolangcilint"
	"go.einride.tech/sage/tools/sggolicenses"
	"go.einride.tech/sage/tools/sggoreleaser"
	"go.einride.tech/sage/tools/sggosemanticrelease"
	"go.einride.tech/sage/tools/sgmdformat"
	"go.einride.tech/sage/tools/sgyamlfmt"
)

func main() {
	sg.GenerateMakefiles(
		sg.Makefile{
			Path:          sg.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}

func All(ctx context.Context) error {
	sg.Deps(ctx, ConvcoCheck, BufLint, FormatMarkdown, FormatYAML)
	sg.Deps(ctx, BufGenerate)
	sg.Deps(ctx, BufGenerateExample)
	sg.Deps(ctx, GoLint, GoTest)
	sg.Deps(ctx, GoModTidy)
	sg.Deps(ctx, GoLicenses)
	sg.Deps(ctx, GitVerifyNoDiff)
	return nil
}

func BufLint(ctx context.Context) error {
	sg.Logger(ctx).Println("linting Buf module..")
	return sgbuf.Command(ctx, "lint").Run()
}

func BufPush(ctx context.Context) error {
	sg.Logger(ctx).Println("pushing Buf module..")
	return sgbuf.Command(ctx, "push").Run()
}

func FormatYAML(ctx context.Context) error {
	sg.Logger(ctx).Println("formatting YAML files...")
	return sgyamlfmt.Run(ctx)
}

func GoModTidy(ctx context.Context) error {
	sg.Logger(ctx).Println("tidying Go module files...")
	return sg.Command(ctx, "go", "mod", "tidy", "-v").Run()
}

func GoTest(ctx context.Context) error {
	sg.Logger(ctx).Println("running Go tests...")
	return sggo.TestCommand(ctx).Run()
}

func GoLint(ctx context.Context) error {
	sg.Logger(ctx).Println("linting Go files...")
	return sggolangcilint.Run(ctx)
}

func GoLicenses(ctx context.Context) error {
	sg.Logger(ctx).Println("checking Go licenses...")
	return sggolicenses.Check(ctx)
}

func FormatMarkdown(ctx context.Context) error {
	sg.Logger(ctx).Println("formatting Markdown files...")
	return sgmdformat.Command(ctx).Run()
}

func ConvcoCheck(ctx context.Context) error {
	sg.Logger(ctx).Println("checking git commits...")
	return sgconvco.Command(ctx, "check", "origin/master..HEAD").Run()
}

func GitVerifyNoDiff(ctx context.Context) error {
	sg.Logger(ctx).Println("verifying that git has no diff...")
	return sggit.VerifyNoDiff(ctx)
}

func ProtocGenGo(ctx context.Context) error {
	sg.Logger(ctx).Println("installing...")
	_, err := sgtool.GoInstallWithModfile(
		ctx,
		"google.golang.org/protobuf/cmd/protoc-gen-go",
		sg.FromGitRoot("go.mod"),
	)
	return err
}

func BufGenerate(ctx context.Context) error {
	sg.Deps(ctx, ProtocGenGo)
	sg.Logger(ctx).Println("generating proto stubs...")
	return sgbuf.Command(
		ctx, "generate", "--output", sg.FromGitRoot(), "--template", "buf.gen.yaml", "--path", "einride",
	).Run()
}

func BufGenerateExample(ctx context.Context) error {
	sg.Deps(ctx, ProtocGenGoGrpcServiceConfig)
	sg.Logger(ctx).Println("generating example...")
	if err := os.RemoveAll(sg.FromGitRoot("internal", "gen", "proto")); err != nil {
		return err
	}
	return sgbuf.Command(
		ctx,
		"generate",
		"--template",
		"buf.gen.example.yaml",
		"--path",
		"einride/serviceconfig/example",
	).Run()
}

func ProtocGenGoGrpcServiceConfig(ctx context.Context) error {
	sg.Logger(ctx).Println("building binary...")
	return sg.Command(
		ctx,
		"go",
		"build",
		"-o",
		sg.FromBinDir("protoc-gen-go-grpc-service-config"),
		"./cmd/protoc-gen-go-grpc-service-config",
	).Run()
}

func SemanticRelease(ctx context.Context, repo string, dry bool) error {
	sg.Logger(ctx).Println("triggering release...")
	args := []string{
		"--allow-initial-development-versions",
		"--allow-no-changes",
		"--ci-condition=default",
		"--provider=github",
		"--provider-opt=slug=" + repo,
	}
	if dry {
		args = append(args, "--dry")
	}
	return sggosemanticrelease.Command(ctx, args...).Run()
}

func GoReleaser(ctx context.Context, snapshot bool) error {
	sg.Logger(ctx).Println("building Go binary releases...")
	if err := sggit.Command(ctx, "fetch", "--force", "--tags").Run(); err != nil {
		return err
	}
	args := []string{
		"release",
		"--clean",
	}
	if len(sggit.Tags(ctx)) == 0 && !snapshot {
		sg.Logger(ctx).Printf("no git tag found for %s, forcing snapshot mode", sggit.ShortSHA(ctx))
		snapshot = true
	}
	if snapshot {
		args = append(args, "--snapshot")
	}
	return sggoreleaser.Command(ctx, args...).Run()
}
