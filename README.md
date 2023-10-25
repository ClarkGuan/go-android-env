# go-android-env

辅助设置 Golang 的 Android NDK 环境变量，简化 Android 交叉编译和运行的流程。

安装后会生成四个可执行文件，分别对应四种交叉编译环境：

* 386：32 位 x86 设备或模拟器
* amd64：64 位 amd64 设备或模拟器
* arm：32 位 armv7a 设备或模拟器
* arm64：64 位 armv8a 设备或模拟器

它们会安装到 `$GOBIN` 或 `$GOPATN/bin` 目录下。请将上述目录添加到 `$PATH` 中以便系统定位和运行它们。

## 依赖

* Golang 环境并设置 $GOPATH 环境变量
* arun 工具（ https://github.com/ClarkGuan/arun ）
* NDK 并设置 `$NDK` 环境变量
* Android 模拟器或真实设备

## 安装

```bash
git clone https://github.com/ClarkGuan/go-android-env
cd go-android-env
./build.sh
```

或

```bash
go install -x github.com/ClarkGuan/go-android-env/...@latest
```


## 使用

### 运行 go run

```bash
[386|amd64|arm|arm64] go run -buildmode=pie -exec=arun <package or directory> [arguments...]
```

举例：

新建 demo 文件夹 `mkdir demo`，初始化 go module `go mod init demo_android`， 新建 main.go 如下：

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello android device!")
	os.Exit(1)  // 强制返回错误码 1
}
```

确保 arun 工具、adb 工具和 NDK 都已经安装完毕。设置 $NDK 环境变量指向 NDK 根目录。

此时分情况：

* 如果需要安装到 Android 模拟器上，请运行

```bash
386 go run -buildmode=pie -exec=arun ./demo
```

* 如果需要在 Android 真实设备上运行（armv7a 或 armv8），请运行

```bash
arm go run -buildmode=pie -exec=arun ./demo
```

或

```bash
arm64 go run -buildmode=pie -exec=arun ./demo
```

如果 Android 设备比较新的话（一定是支持 64 位芯片的）。
如果不熟悉 go run 命令可以查看 `go help run` 文档。
一切顺利的话，会有类似下面的输出：

```text
prepare to push /var/folders/n5/26_m50tn4_j2n3gmvt19300h0000gn/T/go-build440104086/b001/exe/demo to device
/var/folders/n5/26_m50tn4_j2n3gmvt19300h0000gn/T/go-build440104086/b001/exe/demo: 1 file pushed, 0 skipped. 60.2 MB/s (1850264 bytes in 0.029s)
[程序输出如下]
Hello android device!
[程序执行返回错误码(1)]
```

### 运行 go test

和运行 go run 类似，

```bash
[386|amd64|arm|arm64] go test -buildmode=pie -exec=arun <package or directory> [build/test flags & test binary flags]
```

例如，新建单元测试文件 demo_test.go

```go
package main

import "testing"

func TestDemo(t *testing.T) {
	t.Log("Hello Android Device!!!")
	t.Fatal("Force fail.")
}
```

然后在 Android 真实设备中运行该测试 case：

```bash
arm64 go test -buildmode=pie -exec=arun . -v
```

会有如下输出：

```text
prepare to push /var/folders/n5/26_m50tn4_j2n3gmvt19300h0000gn/T/go-build199324000/b001/demo.test to device
/var/folders/n5/26_m50tn4_j2n3gmvt19300h0000gn/T/go-build199324000/b001/demo.test: 1 file pushed, 0 skipped. 76.3 MB/s (2853464 bytes in 0.036s)
[程序输出如下]
=== RUN   TestDemo
    TestDemo: demo_test.go:6: Hello Android Device!!!
    TestDemo: demo_test.go:7: Force fail.
--- FAIL: TestDemo (0.00s)
FAIL
[程序执行返回错误码(1)]
```

### 加入 NDK 头文件和库文件的定位方法

我们经常需要使用 NDK 的一些头文件，比如 `jni.h`、`EGL/egl.h`、`GLES3/gl3.h` 等等。而它们的定位并不是一件容易的事，通过 `gdk` 工具的加入可以帮助我们简化这一过程。我们以调用 NDK 函数输出日志到 logcat 中为例：

```go
package main

//
// #cgo LDFLAGS: -llog
//
// #include <android/log.h>
// #include <stdlib.h>
//
import "C"

import (
	"fmt"
	"unsafe"
)

type LogPriority int

const (
	ANDROID_LOG_UNKNOWN LogPriority = iota
	ANDROID_LOG_DEFAULT
	ANDROID_LOG_VERBOSE
	ANDROID_LOG_DEBUG
	ANDROID_LOG_INFO
	ANDROID_LOG_WARN
	ANDROID_LOG_ERROR
	ANDROID_LOG_FATAL
	ANDROID_LOG_SILENT
)

func write(prio LogPriority, tag, msg string) {
	ctag := C.CString(tag)
	cmsg := C.CString(msg)
	C.__android_log_write(C.int(prio), ctag, cmsg)
	C.free(unsafe.Pointer(ctag))
	C.free(unsafe.Pointer(cmsg))
}

func LogV(tag, format string, a ...any) {
	write(ANDROID_LOG_VERBOSE, tag, fmt.Sprintf(format, a...))
}

func LogD(tag, format string, a ...any) {
	write(ANDROID_LOG_DEBUG, tag, fmt.Sprintf(format, a...))
}

func LogI(tag, format string, a ...any) {
	write(ANDROID_LOG_INFO, tag, fmt.Sprintf(format, a...))
}

func LogW(tag, format string, a ...any) {
	write(ANDROID_LOG_WARN, tag, fmt.Sprintf(format, a...))
}

func LogE(tag, format string, a ...any) {
	write(ANDROID_LOG_ERROR, tag, fmt.Sprintf(format, a...))
}

func main() {
	LogV("clark", "hello from verbose logs")
	LogD("clark", "hello from debug logs")
	LogI("clark", "hello from info logs")
	LogW("clark", "hello from warning logs")
	LogE("clark", "hello from error logs")
}
```

我们打开 terminal 并运行：

```shell
> adb logcat -s clark
```

然后再打开另一个 terminal 编译我们的程序并运行：

```shell
> arm64 gdk go run -exec=arun .   
============================
[exit status:(0)]
    0m00.04s real     0m00.01s user     0m00.03s system
```

然后在第一个 terminal 窗口会看到输出：

```shell
10-25 15:58:22.938 12362 12362 V clark   : hello from verbose logs
10-25 15:58:22.939 12362 12362 D clark   : hello from debug logs
10-25 15:58:22.939 12362 12362 I clark   : hello from info logs
10-25 15:58:22.939 12362 12362 W clark   : hello from warning logs
10-25 15:58:22.939 12362 12362 E clark   : hello from error logs
```

我们只需要添加连接选项 `#cgo LDFLAGS: -llog` 而无须考虑头文件和库文件在文件系统中的具体位置在哪里。

> gdk 具体使用方法：
> 
> `[386|amd64|arm|arm64] [gdk [-level <android API level>]] go ...`
> 
> - 其中 `gdk` 必须出现在 `386|amd64|arm|arm64` 它们任一个之后，其他 `go` 命令之前
> - 可以使用 `-level <android API level>` 指定具体的 API level，默认为 21

