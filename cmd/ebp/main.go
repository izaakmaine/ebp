package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/PithomLabs/ebp/internal/service"
	"github.com/PithomLabs/ebp/internal/store"
)

func main() {
	dbPath := flag.String("db", "ebp.db", "SQLite database path")
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ebp := service.New(db)
	ctx := context.Background()

	switch flag.Arg(0) {
	case "capture":
		cmdCapture(ctx, ebp, flag.Args()[1:])
	case "status":
		cmdStatus(ctx, ebp, flag.Args()[1:])
	case "retire":
		cmdRetire(ctx, ebp, flag.Args()[1:])
	case "add-debt":
		cmdAddDebt(ctx, ebp, flag.Args()[1:])
	case "promote":
		cmdPromote(ctx, ebp, flag.Args()[1:])
	case "mark-final-truth-claim":
		cmdMarkFinalTruth(ctx, ebp, flag.Args()[1:])
	case "clear-final-truth-claim":
		cmdClearFinalTruth(ctx, ebp, flag.Args()[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", flag.Arg(0))
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: ebp [options] <command> [args]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  capture <owner> <claim> [--source <source>]")
	fmt.Fprintln(os.Stderr, "  status <idea-id>")
	fmt.Fprintln(os.Stderr, "  retire <idea-id> <debt-item> <evidence>")
	fmt.Fprintln(os.Stderr, "  add-debt <idea-id> <debt-item>")
	fmt.Fprintln(os.Stderr, "  promote <idea-id>")
	fmt.Fprintln(os.Stderr, "  mark-final-truth-claim <idea-id>")
	fmt.Fprintln(os.Stderr, "  clear-final-truth-claim <idea-id>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Debt items: needMap, needInvariant, needToyCheck, needNullModel, needObstruction, needFaithfulnessReview")
}

func cmdCapture(ctx context.Context, ebp *service.EBP, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ebp capture <owner> <claim> [--source <source>]")
		os.Exit(1)
	}
	owner := args[0]
	claim := args[1]
	source := ""
	for i := 2; i < len(args)-1; i++ {
		if args[i] == "--source" {
			source = args[i+1]
			i++
		}
	}

	idea, err := ebp.Capture(ctx, owner, claim, source, owner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Idea captured.\n")
	fmt.Printf("ID: %s\n", idea.ID)
	fmt.Printf("Status: alive, unpromoted.\n")
	fmt.Println("Debt:")
	for _, d := range []string{"needMap", "needInvariant", "needToyCheck", "needNullModel", "needObstruction", "needFaithfulnessReview"} {
		fmt.Printf("  - %s\n", d)
	}
}

func cmdStatus(ctx context.Context, ebp *service.EBP, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ebp status <idea-id>")
		os.Exit(1)
	}

	idea, debt, err := ebp.Status(ctx, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Alive: yes.\n")
	if idea.Promoted {
		fmt.Printf("Promoted: yes.\n")
		fmt.Printf("Promoted at: %s.\n", idea.PromotedAt)
	} else {
		fmt.Printf("Promoted: no.\n")
	}
	if idea.ContainsFinalTruthClaim {
		fmt.Printf("Final-truth claim: yes.\n")
	} else {
		fmt.Printf("Final-truth claim: no.\n")
	}

	openCount := 0
	fmt.Println("Debt:")
	for _, d := range debt {
		if !d.Retired {
			fmt.Printf("  - %s\n", d.Item)
			openCount++
		}
	}
	if openCount == 0 {
		fmt.Println("  (none)")
	}
}

func cmdRetire(ctx context.Context, ebp *service.EBP, args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ebp retire <idea-id> <debt-item> <evidence>")
		os.Exit(1)
	}

	if err := ebp.Retire(ctx, args[0], args[1], args[2], "cli"); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Debt retired: %s.\n", args[1])
}

func cmdAddDebt(ctx context.Context, ebp *service.EBP, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ebp add-debt <idea-id> <debt-item>")
		os.Exit(1)
	}

	if err := ebp.AddDebt(ctx, args[0], args[1], "cli"); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Debt added: %s.\n", args[1])
}

func cmdPromote(ctx context.Context, ebp *service.EBP, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ebp promote <idea-id>")
		os.Exit(1)
	}

	if err := ebp.Promote(ctx, args[0], "cli"); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Idea promoted.")
}

func cmdMarkFinalTruth(ctx context.Context, ebp *service.EBP, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ebp mark-final-truth-claim <idea-id>")
		os.Exit(1)
	}

	if err := ebp.MarkFinalTruthClaim(ctx, args[0], "cli"); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Final-truth claim flagged.")
}

func cmdClearFinalTruth(ctx context.Context, ebp *service.EBP, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ebp clear-final-truth-claim <idea-id>")
		os.Exit(1)
	}

	if err := ebp.ClearFinalTruthClaim(ctx, args[0], "cli"); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Final-truth claim cleared.")
}
