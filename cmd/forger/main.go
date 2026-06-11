package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kn-ll/forger/internal/thread"
)

func main() {
	// os.Args 是 Go 程序启动时收到的命令行参数切片。
	// os.Args[1:] 就是去掉程序名，只把真正的命令参数传给 run
	os.Exit(run(context.Background(), os.Args[1:]))
}

// run 是 CLI 的顶层分发入口。当前阶段只开放 thread 命令，后续所有能力也应从
// thread runtime 进入，避免重新长出彼此独立的 run/code/admin 编排。
func run(ctx context.Context, args []string) int {
	store := thread.NewFileStore(".forger")
	if len(args) == 0 {
		printHelp()
		return 0
	}
	switch args[0] {
	case "thread":
		return runThread(ctx, store, args[1:])
	case "help", "-h", "--help":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		printHelp()
		return 2
	}
}

// runThread 处理线程相关命令。线程是 Forger 的第一产品对象，所有 Agent
// 执行、工具调用、审批和产物最终都应该挂回某个 thread。
func runThread(ctx context.Context, store thread.Store, args []string) int {
	if len(args) == 0 {
		printThreadHelp()
		return 0
	}
	switch args[0] {
	case "new":
		title := strings.TrimSpace(strings.Join(args[1:], " "))
		if title == "" {
			fmt.Fprintln(os.Stderr, "thread title is required")
			return 2
		}
		created, err := store.Create(ctx, thread.CreateRequest{Title: title})
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		fmt.Printf("%s\t%s\n", created.ID, created.Title)
		return 0
	case "list":
		threads, err := store.List(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		for _, item := range threads {
			fmt.Printf("%s\t%s\t%s\n", item.ID, item.Status, item.Title)
		}
		return 0
	case "show":
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			fmt.Fprintln(os.Stderr, "thread id is required")
			return 2
		}
		item, err := store.Get(ctx, strings.TrimSpace(args[1]))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		fmt.Printf("Thread\t%s\n", item.ID)
		fmt.Printf("Title\t%s\n", item.Title)
		fmt.Printf("Status\t%s\n", item.Status)
		fmt.Printf("Created\t%s\n", item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
		fmt.Printf("Updated\t%s\n", item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
		fmt.Printf("Messages\t%d\n", len(item.Messages))
		for _, message := range item.Messages {
			fmt.Printf("message\t%s\t%s\t%s\n", message.ID, message.Role, strings.ReplaceAll(message.Content, "\n", "\\n"))
		}
		fmt.Printf("Runs\t%d\n", len(item.Runs))
		for _, run := range item.Runs {
			fmt.Printf("run\t%s\t%s\t%s\n", run.ID, run.Status, run.Goal)
		}
		fmt.Printf("ToolCalls\t%d\n", len(item.ToolCalls))
		for _, call := range item.ToolCalls {
			fmt.Printf("toolcall\t%s\t%s\t%s\n", call.ID, call.Status, call.Tool)
		}
		fmt.Printf("Artifacts\t%d\n", len(item.Artifacts))
		for _, artifact := range item.Artifacts {
			fmt.Printf("artifact\t%s\t%s\t%s\n", artifact.ID, artifact.Kind, artifact.Title)
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown thread command: %s\n", args[0])
		printThreadHelp()
		return 2
	}
}

func printHelp() {
	fmt.Println("Forger")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  forger thread new <title>")
	fmt.Println("  forger thread list")
	fmt.Println("  forger thread show <thread-id>")
}

func printThreadHelp() {
	fmt.Println("Usage:")
	fmt.Println("  forger thread new <title>")
	fmt.Println("  forger thread list")
	fmt.Println("  forger thread show <thread-id>")
}
