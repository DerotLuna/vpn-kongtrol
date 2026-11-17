// Command impactcheck reports which user-facing surfaces a change may have
// left out of date, from docs/impact.yml. Advisory only: it always exits 0
// unless it cannot run at all.
//
//	go run ./internal/impact/impactcheck -base origin/main -head HEAD \
//	    -out comment.md
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/impact"
)

func main() {
	base := flag.String("base", "origin/main", "base ref")
	head := flag.String("head", "HEAD", "head ref")
	out := flag.String("out", "", "write the Markdown report here (default stdout)")
	flag.Parse()

	m, err := impact.Load(".")
	if err != nil {
		fail(err)
	}
	names, err := git("diff", "--name-only", *base+"..."+*head)
	if err != nil {
		fail(err)
	}
	changed := strings.Fields(names)
	hits := m.Analyze(changed)
	problems := m.Problems(".")
	report := impact.Markdown(hits, problems)
	if *out != "" {
		if err := os.WriteFile(*out, []byte(report), 0o644); err != nil {
			fail(err)
		}
	} else {
		fmt.Print(report)
	}

	pending := 0
	for _, h := range hits {
		pending += len(h.Untouched)
		for _, s := range h.Untouched {
			annotate(fmt.Sprintf("%s changed but %s did not — update it or say why not (no aplica: <motivo>)", h.Feature, s))
		}
	}
	for _, p := range problems {
		annotate(p)
	}
	if gh := os.Getenv("GITHUB_OUTPUT"); gh != "" {
		if f, err := os.OpenFile(gh, os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintf(f, "pending=%d\n", pending)
			f.Close()
		}
	}

}

func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return string(out), err
}

func annotate(msg string) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fmt.Fprintf(os.Stderr, "::warning title=Docs impact::%s\n", msg)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "impactcheck:", err)
	os.Exit(1)
}
