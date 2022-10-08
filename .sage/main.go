package main

import (
	"context"
	"os"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgbuf"
	"go.einride.tech/sage/tools/sgconvco"
	"go.einride.tech/sage/tools/sggit"
	"go.einride.tech/sage/tools/sggo"
	"go.einride.tech/sage/tools/sggolangcilint"
	"go.einride.tech/sage/tools/sggoreview"
	"go.einride.tech/sage/tools/sgmarkdownfmt"
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
	sg.Deps(ctx, ConvcoCheck, GoLint, GoReview, GoTest, BufLint, FormatMarkdown, FormatYAML)
	sg.Deps(ctx, BufGenerateExample)
	sg.SerialDeps(ctx, GoModTidy, GitVerifyNoDiff)
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

func GoReview(ctx context.Context) error {
	sg.Logger(ctx).Println("reviewing Go files...")
	return sggoreview.Command(ctx, "-c", "1", "./...").Run()
}

func GoLint(ctx context.Context) error {
	sg.Logger(ctx).Println("linting Go files...")
	return sggolangcilint.Run(ctx)
}

func FormatMarkdown(ctx context.Context) error {
	sg.Logger(ctx).Println("formatting Markdown files...")
	return sgmarkdownfmt.Command(ctx, "-w", ".").Run()
}

func ConvcoCheck(ctx context.Context) error {
	sg.Logger(ctx).Println("checking git commits...")
	return sgconvco.Command(ctx, "check", "origin/master..HEAD").Run()
}

func GitVerifyNoDiff(ctx context.Context) error {
	sg.Logger(ctx).Println("verifying that git has no diff...")
	return sggit.VerifyNoDiff(ctx)
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
