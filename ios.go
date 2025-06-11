package env

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const sh = "#!/bin/sh\n\n# go/clangwrap.sh\n\nSDK_PATH=`xcrun --sdk $SDK --show-sdk-path`\nCLANG=`xcrun --sdk $SDK --find clang`\n\nif [ \"$GOARCH\" == \"amd64\" ]; then\n    CARCH=\"x86_64\"\nelif [ \"$GOARCH\" == \"arm64\" ]; then\n    CARCH=\"arm64\"\nfi\n\nexec $CLANG -arch $CARCH -isysroot $SDK_PATH -mios-version-min=10.0 \"$@\""

var home, _ = os.UserHomeDir()

var shPath = filepath.Join(home, ".clang-ios-go-wrap.sh")

func prepare() error {
	_, err := os.Stat(shPath)
	if os.IsNotExist(err) {
		if err := os.WriteFile(shPath, []byte(sh), 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	info, err := os.Stat(shPath)
	if err != nil {
		return err
	}

	if info.Mode().Perm() != 0755 {
		if err := os.Chmod(shPath, 0755); err != nil {
			return err
		}
	}

	return nil
}

var iosEnvs = map[string][]string{
	"amd64": {
		"GOOS=ios",
		"GOARCH=amd64",
		"SDK=iphonesimulator",
		fmt.Sprintf("CC=%s", shPath),
		"CGO_CFLAGS=-fembed-bitcode",
	},
	"arm64": {
		"GOOS=ios",
		"GOARCH=arm64",
		"SDK=iphoneos",
		fmt.Sprintf("CC=%s", shPath),
		"CGO_CFLAGS=-fembed-bitcode",
	},
}

func IosMain(arch string) {
	if !xcodeAvailable() {
		log.Fatalln(fmt.Errorf("xcode not available"))
	}

	//if err := prepare(); err != nil {
	//	log.Fatalln(err)
	//}

	//envs := iosEnvs[arch]
	//if len(envs) == 0 {
	//	log.Fatalln(fmt.Errorf("%s not supported by clang", arch))
	//}

	if err := iosEnvSetup(); err != nil {
		log.Fatalln(err)
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, appleEnv[arch]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

var (
	appleEnv        map[string][]string
	appleNM         string
	buildIOSVersion = "15.0"
	buildN          = false
)

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func isApplePlatform(platform string) bool {
	return contains(applePlatforms, platform)
}

var applePlatforms = []string{"ios", "iossimulator", "macos", "maccatalyst"}

func xcodeAvailable() bool {
	err := exec.Command("xcrun", "xcodebuild", "-version").Run()
	return err == nil
}

func platformArchs(platform string) []string {
	switch platform {
	case "ios":
		return []string{"arm64"}
	case "iossimulator":
		return []string{"arm64", "amd64"}
	case "macos", "maccatalyst":
		return []string{"arm64", "amd64"}
	default:
		panic(fmt.Sprintf("unexpected platform: %s", platform))
	}
}

func isSupportedArch(platform, arch string) bool {
	return contains(platformArchs(platform), arch)
}

func platformOS(platform string) string {
	switch platform {
	case "ios", "iossimulator":
		return "ios"
	case "macos", "maccatalyst":
		// For "maccatalyst", Go packages should be built with GOOS=darwin,
		// not GOOS=ios, since the underlying OS (and kernel, runtime) is macOS.
		// But, using GOOS=darwin with build-tag ios leads to corrupt builds: https://go.dev/issue/52299
		// => So we use GOOS=ios for now.
		// We also apply a "macos" or "maccatalyst" build tag, respectively.
		// See below for additional context.
		return "ios"
	default:
		panic(fmt.Sprintf("unexpected platform: %s", platform))
	}
}

func platformTags(platform string) []string {
	switch platform {
	case "ios", "iossimulator":
		return []string{"ios"}
	case "macos":
		return []string{"macos"}
	case "maccatalyst":
		// Mac Catalyst is a subset of iOS APIs made available on macOS
		// designed to ease porting apps developed for iPad to macOS.
		// See
		//   https://developer.apple.com/mac-catalyst/.
		//   https://stackoverflow.com/questions/12132933/preprocessor-macro-for-os-x-targets/49560690#49560690
		//
		// Historically gomobile used GOOS=darwin with build tag ios when
		// targeting Mac Catalyst. However, this configuration is not officially
		// supported and leads to corrupt builds after go1.18: https://go.dev/issues/52299
		// Use GOOS=ios.
		// To help discriminate between darwin, ios, macos, and maccatalyst
		// targets, there is also a "maccatalyst" tag.
		return []string{"macos", "maccatalyst"}
	default:
		panic(fmt.Sprintf("unexpected platform: %s", platform))
	}
}

func envClang(sdkName string) (clang, cflags string, err error) {
	if buildN {
		return sdkName + "-clang", "-isysroot " + sdkName, nil
	}
	cmd := exec.Command("xcrun", "--sdk", sdkName, "--find", "clang")
	out, err := cmd.Output()
	if err != nil {
		if ee := (*exec.ExitError)(nil); errors.As(err, &ee) {
			out = append(out, ee.Stderr...)
		}
		return "", "", fmt.Errorf("xcrun --find: %v\n%s", err, out)
	}
	clang = strings.TrimSpace(string(out))

	cmd = exec.Command("xcrun", "--sdk", sdkName, "--show-sdk-path")
	out, err = cmd.Output()
	if err != nil {
		if ee := (*exec.ExitError)(nil); errors.As(err, &ee) {
			out = append(out, ee.Stderr...)
		}
		return "", "", fmt.Errorf("xcrun --show-sdk-path: %v\n%s", err, out)
	}
	sdk := strings.TrimSpace(string(out))
	return clang, "-isysroot " + sdk, nil
}

func archClang(goarch string) string {
	switch goarch {
	case "arm":
		return "armv7"
	case "arm64":
		return "arm64"
	case "386":
		return "i386"
	case "amd64":
		return "x86_64"
	default:
		panic(fmt.Sprintf("unknown GOARCH: %q", goarch))
	}
}

func iosEnvSetup() error {
	appleNM = "nm"
	appleEnv = make(map[string][]string)
	for _, platform := range applePlatforms {
		for _, arch := range platformArchs(platform) {
			var env []string
			var goos, sdk, clang, cflags string
			var err error
			switch platform {
			case "ios":
				goos = "ios"
				sdk = "iphoneos"
				clang, cflags, err = envClang(sdk)
				cflags += " -miphoneos-version-min=" + buildIOSVersion
				//cflags += " -fembed-bitcode"
			case "iossimulator":
				goos = "ios"
				sdk = "iphonesimulator"
				clang, cflags, err = envClang(sdk)
				cflags += " -mios-simulator-version-min=" + buildIOSVersion
				//cflags += " -fembed-bitcode"
			case "maccatalyst":
				// See the comment about maccatalyst's GOOS, build tags configuration
				// in platformOS and platformTags.
				// Using GOOS=darwin with build-tag ios leads to corrupt builds: https://go.dev/issue/52299
				// => So we use GOOS=ios for now.
				goos = "ios"
				sdk = "macosx"
				clang, cflags, err = envClang(sdk)
				// TODO(ydnar): the following 3 lines MAY be needed to compile
				// packages or apps for maccatalyst. Commenting them out now in case
				// it turns out they are necessary. Currently none of the example
				// apps will build for macos or maccatalyst because they have a
				// GLKit dependency, which is deprecated on all Apple platforms, and
				// broken on maccatalyst (GLKView isn’t available).
				// sysroot := strings.SplitN(cflags, " ", 2)[1]
				// cflags += " -isystem " + sysroot + "/System/iOSSupport/usr/include"
				// cflags += " -iframework " + sysroot + "/System/iOSSupport/System/Library/Frameworks"
				switch arch {
				case "amd64":
					cflags += " -target x86_64-apple-ios" + buildIOSVersion + "-macabi"
				case "arm64":
					cflags += " -target arm64-apple-ios" + buildIOSVersion + "-macabi"
					//cflags += " -fembed-bitcode"
				}
			case "macos":
				goos = "darwin"
				sdk = "macosx" // Note: the SDK is called "macosx", not "macos"
				clang, cflags, err = envClang(sdk)
				if arch == "arm64" {
					//cflags += " -fembed-bitcode"
				}
			default:
				panic(fmt.Errorf("unknown Apple target: %s/%s", platform, arch))
			}

			if err != nil {
				return err
			}

			env = append(env,
				"GOOS="+goos,
				"GOARCH="+arch,
				"GOFLAGS="+"-tags="+strings.Join(platformTags(platform), ","),
				"CC="+clang,
				"CXX="+clang+"++",
				"CGO_CFLAGS="+cflags+" -arch "+archClang(arch),
				"CGO_CXXFLAGS="+cflags+" -arch "+archClang(arch),
				"CGO_LDFLAGS="+cflags+" -arch "+archClang(arch),
				"CGO_ENABLED=1",
				"DARWIN_SDK="+sdk,
			)
			appleEnv[platform+"/"+arch] = env
		}
	}
	return nil
}
