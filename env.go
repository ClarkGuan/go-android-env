package env

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const minAndroidAPI = 15

// TODO 可以手动修改？
var buildAndroidAPI = minAndroidAPI

var androidEnv map[string][]string // android arch -> []string

type ndkToolchain struct {
	arch        string
	abi         string
	minAPI      int
	toolPrefix  string
	clangPrefix string
}

func (tc *ndkToolchain) ClangPrefix() string {
	if buildAndroidAPI < tc.minAPI {
		return fmt.Sprintf("%s%d", tc.clangPrefix, tc.minAPI)
	}
	return fmt.Sprintf("%s%d", tc.clangPrefix, buildAndroidAPI)
}

func archNDK() string {
	if runtime.GOOS == "windows" && runtime.GOARCH == "386" {
		return "windows"
	} else {
		var arch string
		switch runtime.GOARCH {
		case "386":
			arch = "x86"
		case "amd64":
			arch = "x86_64"
		default:
			panic("unsupported GOARCH: " + runtime.GOARCH)
		}
		return runtime.GOOS + "-" + arch
	}
}

func (tc *ndkToolchain) Path(ndkRoot, toolName string) string {
	var pref string
	switch toolName {
	case "clang", "clang++":
		pref = tc.ClangPrefix()
	default:
		pref = tc.toolPrefix
	}
	return filepath.Join(ndkRoot, "toolchains", "llvm", "prebuilt", archNDK(), "bin", pref+"-"+toolName)
}

type ndkConfig map[string]ndkToolchain // map: GOOS->androidConfig.

func (nc ndkConfig) Toolchain(arch string) ndkToolchain {
	tc, ok := nc[arch]
	if !ok {
		panic(`unsupported architecture: ` + arch)
	}
	return tc
}

var ndk = ndkConfig{
	"arm": {
		arch:        "arm",
		abi:         "armeabi-v7a",
		minAPI:      16,
		toolPrefix:  "arm-linux-androideabi",
		clangPrefix: "armv7a-linux-androideabi",
	},
	"arm64": {
		arch:        "arm64",
		abi:         "arm64-v8a",
		minAPI:      21,
		toolPrefix:  "aarch64-linux-android",
		clangPrefix: "aarch64-linux-android",
	},

	"386": {
		arch:        "x86",
		abi:         "x86",
		minAPI:      16,
		toolPrefix:  "i686-linux-android",
		clangPrefix: "i686-linux-android",
	},
	"amd64": {
		arch:        "x86_64",
		abi:         "x86_64",
		minAPI:      21,
		toolPrefix:  "x86_64-linux-android",
		clangPrefix: "x86_64-linux-android",
	},
}

func ndkRoot() (string, error) {
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome != "" {
		ndkRoot := filepath.Join(androidHome, "ndk-bundle")
		_, err := os.Stat(ndkRoot)
		if err == nil {
			return ndkRoot, nil
		}
	}

	ndkPaths := []string{"NDK", "ANDROID_NDK_HOME"}
	ndkRoot := ""
	for _, path := range ndkPaths {
		ndkRoot = os.Getenv(path)
		if ndkRoot != "" {
			_, err := os.Stat(ndkRoot)
			if err == nil {
				return ndkRoot, nil
			}
		}
	}

	return "", fmt.Errorf("no Android NDK found in $ANDROID_HOME/ndk-bundle nor in $ANDROID_NDK_HOME")
}

func envInit() error {
	// Setup the cross-compiler environments.
	if ndkRoot, err := ndkRoot(); err == nil {
		androidEnv = make(map[string][]string)
		if buildAndroidAPI < minAndroidAPI {
			return fmt.Errorf("go-android-env requires Android API level >= %d", minAndroidAPI)
		}
		for arch, toolchain := range ndk {
			clang := toolchain.Path(ndkRoot, "clang")
			clangpp := toolchain.Path(ndkRoot, "clang++")
			tools := []string{clang, clangpp}
			if runtime.GOOS == "windows" {
				// Because of https://github.com/android-ndk/ndk/issues/920,
				// we require r19c, not just r19b. Fortunately, the clang++.cmd
				// script only exists in r19c.
				tools = append(tools, clangpp+".cmd")
			}
			for _, tool := range tools {
				_, err = os.Stat(tool)
				if err != nil {
					return fmt.Errorf("No compiler for %s was found in the NDK (tried %s). Make sure your NDK version is >= r19c. Use `sdkmanager --update` to update it.", arch, tool)
				}
			}
			androidEnv[arch] = []string{
				"GOOS=android",
				"GOARCH=" + arch,
				"CC=" + clang,
				"CXX=" + clangpp,
				"CGO_ENABLED=1",
			}
			if arch == "arm" {
				androidEnv[arch] = append(androidEnv[arch], "GOARM=7")
			}
		}
	}
	return nil
}

func Main(s string) {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "缺少参数")
		os.Exit(1)
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, androidEnv["386"]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func init() {
	err := envInit()
	if err != nil {
		panic(err)
	}
}
