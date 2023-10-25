package env

import (
	"fmt"
	"os"
	"os/exec"
)

func LibsMain(level int, args ...string) {
	ndkRoot, err := envInit()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	arch := os.Getenv("GOARCH")
	if len(arch) == 0 {
		fmt.Fprintf(os.Stderr, "no $GOARCH found\n")
		os.Exit(1)
	}
	toolchain := ndk[arch]

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(cmd.Env, os.Environ()...)

	// CGO_CFLAGS & CGO_CXXFLAGS
	cflags := os.Getenv("CGO_CFLAGS")
	cxxflags := os.Getenv("CGO_CXXFLAGS")
	includePath := toolchain.includePath(ndkRoot)
	cflags = fmt.Sprintf("CGO_CFLAGS=-I%s %s", includePath, cflags)
	cxxflags = fmt.Sprintf("CGO_CXXFLAGS=-I%s %s", includePath, cxxflags)

	// CGO_LDFLAGS
	cLdFlags := os.Getenv("CGO_LDFLAGS")
	libsPath := toolchain.libraryPath(ndkRoot, level)
	if _, err := os.Stat(libsPath); err != nil {
		fmt.Fprintln(os.Stderr, libsPath, ":", err)
		os.Exit(1)
	}
	cLdFlags = fmt.Sprintf("CGO_LDFLAGS=-L%s %s", libsPath, cLdFlags)

	cmd.Env = append(cmd.Env, cflags, cxxflags, cLdFlags)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
