package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/anthro-env/anthro-env/internal/core"
	"github.com/anthro-env/anthro-env/internal/ui"
	"golang.org/x/term"
)

var version = "0.1.9"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	mgr := core.NewManager()
	if len(args) == 0 {
		return runMenu(mgr)
	}

	switch args[0] {
	case "-v", "--version", "version":
		fmt.Printf("anthro-env %s\n", version)
		return nil
	case "migrate-tokens":
		return runMigrateTokens(mgr)
	case "menu":
		return runMenu(mgr)
	case "init":
		return runInit(mgr)
	case "add", "edit", "use", "ls", "current", "rm":
		return runProfile(mgr, args)
	case "doctor":
		return runDoctor(mgr)
	case "hook":
		if len(args) < 2 {
			return errors.New("usage: anthro-env hook <zsh|bash>")
		}
		fmt.Print(core.HookScript(args[1]))
		return nil
	case "env", "export":
		return runExport(mgr)
	case "profile":
		// Backward compatibility for older syntax.
		return runProfile(mgr, args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runInit(mgr *core.Manager) error {
	if err := mgr.EnsureLayout(); err != nil {
		return err
	}

	shell := core.DetectShell(os.Getenv("SHELL"))
	rcFile := core.RCFile(shell)
	if rcFile == "" {
		return fmt.Errorf("unsupported shell: %s", shell)
	}
	if err := core.InstallHook(rcFile, shell); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Anthro Env initialization")
	fmt.Println("Tip: ANTHROPIC_MODEL is optional. Leave empty to use provider/gateway default model.")
	fmt.Print("Profile name [default]: ")
	name, err := readInputLine(reader)
	if err != nil {
		return err
	}
	if name == "" {
		name = "default"
	}
	if !core.ValidProfileName(name) {
		return errors.New("invalid profile name: use letters, numbers, -, _")
	}

	fmt.Print("ANTHROPIC_BASE_URL: ")
	baseURL, err := readInputLine(reader)
	if err != nil {
		return err
	}

	fmt.Print("ANTHROPIC_MODEL (optional, press Enter to skip): ")
	model, err := readInputLine(reader)
	if err != nil {
		return err
	}

	tokenHint := "stored in Keychain"
	if core.IsSSHSession() {
		tokenHint = "SSH mode: will be stored in plaintext"
	}
	fmt.Printf("API credential (%s, exported as ANTHROPIC_API_KEY for MiniMax and ANTHROPIC_AUTH_TOKEN otherwise): ", tokenHint)
	token, err := readInputLine(reader)
	if err != nil {
		return err
	}

	vars := map[string]string{}
	if baseURL != "" {
		vars["ANTHROPIC_BASE_URL"] = baseURL
	}
	if model != "" {
		vars["ANTHROPIC_MODEL"] = model
		vars["ANTHROPIC_SMALL_FAST_MODEL"] = model
		vars["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
		vars["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
		vars["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
	}

	if err := mgr.SaveProfile(name, vars); err != nil {
		return err
	}
	if token != "" {
		if err := mgr.SaveToken(name, token); err != nil {
			return err
		}
	}
	if err := mgr.UseProfile(name); err != nil {
		return err
	}

	fmt.Printf("Initialized. Active profile: %s\n", name)
	fmt.Printf("Hook installed in: %s\n", rcFile)
	fmt.Printf("Run: source %s\n", rcFile)
	return nil
}

// stdinWouldBlock reports whether err is EAGAIN/EWOULDBLOCK from a non-blocking os.File Read.
// os.Stdin.Read wraps syscall errors in *fs.PathError, so direct == syscall.EAGAIN fails.
func stdinWouldBlock(err error) bool {
	return err != nil && (errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK))
}

func runMenu(mgr *core.Manager) error {
	profiles, err := mgr.ListProfiles()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		fmt.Println("No profiles found. Starting initialization...")
		fmt.Println()
		return runInit(mgr)
	}
	sort.Strings(profiles)
	active, _ := mgr.CurrentProfile()

	activeDisplay := core.OrDefault(active, "none")
	if active != "" {
		activeDisplay = ui.BoldGreen(active)
	}
	fmt.Printf("Current profile: %s\n", activeDisplay)
	fmt.Println("Select a profile (use arrow keys, Enter to select, Esc to exit):")
	fmt.Println()

	// Check if terminal supports interactive mode
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return runMenuNonInteractive(mgr, profiles, active)
	}

	return runMenuInteractive(mgr, profiles, active)
}

func runMenuNonInteractive(mgr *core.Manager, profiles []string, active string) error {
	fmt.Println("[0] Exit")
	for i, p := range profiles {
		model, _ := mgr.ProfileModel(p)
		if model == "" {
			model = "-"
		}
		tag := ""
		if p == active {
			tag = " (current"
			if model != "" {
				tag += ", model: " + model
			}
			tag += ")"
			fmt.Printf("[%d] %s%s\n", i+1, ui.BoldGreen(p), tag)
		} else {
			tag = " (model: " + model + ")"
			fmt.Printf("[%d] %s%s\n", i+1, p, tag)
		}
	}
	fmt.Print("Enter number: ")
	reader := bufio.NewReader(os.Stdin)
	in, err := readInputLine(reader)
	if err != nil {
		return err
	}
	index, err := ui.ParseMenuSelection(in, len(profiles))
	if err != nil {
		return err
	}
	if index == 0 {
		fmt.Println("Canceled")
		return nil
	}

	name := profiles[index-1]
	if err := mgr.UseProfile(name); err != nil {
		return err
	}
	fmt.Printf("Switched to profile: %s\n", name)
	return nil
}

func runMenuInteractive(mgr *core.Manager, profiles []string, active string) error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fall back to non-interactive mode
		return runMenuNonInteractive(mgr, profiles, active)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Set stdin to non-blocking mode
	if err := syscall.SetNonblock(int(os.Stdin.Fd()), true); err != nil {
		return err
	}
	defer syscall.SetNonblock(int(os.Stdin.Fd()), false)

	selected := 0
	// Default selection to current profile (or 0 = Exit if none)
	if active != "" {
		for i, p := range profiles {
			if p == active {
				selected = i + 1
				break
			}
		}
	}
	total := len(profiles) + 1 // +1 for Exit option
	// Cursor for redraw sits on the "Selected:" line below the menu; clear that row plus every menu row.
	menuLines := len(profiles) + 1
	redrawLines := menuLines + 1

	clearLines := func(n int) {
		for i := 0; i < n; i++ {
			fmt.Print("\033[2K\033[1G") // Clear line and go to column 1
			if i < n-1 {
				fmt.Print("\033[A") // Move cursor up
			}
		}
	}

	printMenu := func() {
		fmt.Print("\r\033[K") // Clear from cursor to end of line
		fmt.Println("[0] Exit")
		for i, p := range profiles {
			model, _ := mgr.ProfileModel(p)
			if model == "" {
				model = "-"
			}
			tag := ""
			num := i + 1
			if p == active {
				tag = " (current"
				if model != "" {
					tag += ", model: " + model
				}
				tag += ")"
				if num == selected {
					fmt.Printf("\r[%d] \033[1;32m▶ %s%s\033[0m\n", num, ui.BoldGreen(p), tag)
				} else {
					fmt.Printf("\r[%d] %s%s\n", num, ui.BoldGreen(p), tag)
				}
			} else {
				tag = " (model: " + model + ")"
				if num == selected {
					fmt.Printf("\r[%d] \033[1;32m▶ %s%s\033[0m\n", num, p, tag)
				} else {
					fmt.Printf("\r[%d] %s%s\n", num, p, tag)
				}
			}
		}
	}

	// Initial print (cursor ends on the line below the last profile)
	printMenu()
	fmt.Print("\r\033[K") // clear the blank line after menu
	fmt.Print("Selected: ")

	for {
		b := make([]byte, 1)
		n, err := os.Stdin.Read(b)
		if err != nil {
			if stdinWouldBlock(err) {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return err
		}
		if n == 0 {
			continue
		}

		switch b[0] {
		case 13: // Enter
			clearLines(redrawLines)
			if selected == 0 {
				fmt.Println("Canceled")
				return nil
			}
			name := profiles[selected-1]
			if err := mgr.UseProfile(name); err != nil {
				return err
			}
			fmt.Printf("Switched to profile: %s\n", name)
			return nil
		case 27: // ESC or arrow key start
			// Wait a bit to see if more bytes arrive (arrow key) or not (ESC alone)
			time.Sleep(50 * time.Millisecond)
			b2 := make([]byte, 1)
			n2, err := os.Stdin.Read(b2)
			if n2 == 0 {
				if err != nil && !stdinWouldBlock(err) {
					return err
				}
				clearLines(redrawLines)
				fmt.Println("Canceled")
				return nil
			}
			if err != nil && !stdinWouldBlock(err) {
				return err
			}
			if b2[0] == 91 { // '[' - arrow key sequence continues
				b3 := make([]byte, 1)
				for attempts := 0; attempts < 40; attempts++ {
					n3, err3 := os.Stdin.Read(b3)
					if n3 > 0 {
						break
					}
					if err3 != nil && !stdinWouldBlock(err3) {
						return err3
					}
					time.Sleep(5 * time.Millisecond)
				}
				switch b3[0] {
				case 65: // Up arrow
					if selected > 0 {
						selected--
						clearLines(redrawLines)
						printMenu()
						fmt.Print("\r\033[K")
						fmt.Print("Selected: ")
					}
				case 66: // Down arrow
					if selected < total-1 {
						selected++
						clearLines(redrawLines)
						printMenu()
						fmt.Print("\r\033[K")
						fmt.Print("Selected: ")
					}
				}
			}
		case '0':
			selected = 0
			clearLines(redrawLines)
			printMenu()
			fmt.Print("\r\033[K")
			fmt.Print("Selected: ")
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			num := int(b[0] - '0')
			if num < total {
				selected = num
				clearLines(redrawLines)
				printMenu()
				fmt.Print("\r\033[K")
				fmt.Print("Selected: ")
			}
		}
	}
}

func runProfile(mgr *core.Manager, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: anthro-env <add|edit|use|ls|current|rm>")
	}

	switch args[0] {
	case "ls":
		profiles, err := mgr.ListProfiles()
		if err != nil {
			return err
		}
		sort.Strings(profiles)
		active, _ := mgr.CurrentProfile()
		for _, p := range profiles {
			if p == active {
				fmt.Printf("%s *\n", ui.BoldGreen(p))
			} else {
				fmt.Println(p)
			}
		}
		return nil
	case "current":
		current, err := mgr.CurrentProfile()
		if err != nil || current == "" {
			fmt.Println("none")
			return nil
		}
		fmt.Println(current)
		return nil
	case "use":
		if len(args) < 2 {
			return errors.New("usage: anthro-env use <name>")
		}
		if err := mgr.UseProfile(args[1]); err != nil {
			return err
		}
		fmt.Printf("Switched to profile: %s\n", args[1])
		return nil
	case "add":
		if len(args) < 2 {
			return errors.New("usage: anthro-env add <name>")
		}
		name := args[1]
		if !core.ValidProfileName(name) {
			return errors.New("invalid profile name")
		}
		if err := mgr.EnsureLayout(); err != nil {
			return err
		}
		f := filepath.Join(mgr.ProfilesDir(), name+".env")
		if _, err := os.Stat(f); err == nil {
			return fmt.Errorf("profile exists: %s", name)
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("ANTHROPIC_BASE_URL: ")
		baseURL, err := readInputLine(reader)
		if err != nil {
			return err
		}
		fmt.Println("Tip: ANTHROPIC_MODEL is optional. Leave empty to use provider/gateway default model.")
		fmt.Print("ANTHROPIC_MODEL (optional, press Enter to skip): ")
		model, err := readInputLine(reader)
		if err != nil {
			return err
		}
		addTokenHint := "stored in Keychain"
		if core.IsSSHSession() {
			addTokenHint = "SSH mode: will be stored in plaintext"
		}
		fmt.Printf("API credential (%s, exported as ANTHROPIC_API_KEY for MiniMax and ANTHROPIC_AUTH_TOKEN otherwise): ", addTokenHint)
		token, err := readInputLine(reader)
		if err != nil {
			return err
		}
		vars := map[string]string{}
		if baseURL != "" {
			vars["ANTHROPIC_BASE_URL"] = baseURL
		}
		if model != "" {
			vars["ANTHROPIC_MODEL"] = model
			vars["ANTHROPIC_SMALL_FAST_MODEL"] = model
			vars["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
			vars["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
			vars["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
		}
		if err := mgr.SaveProfile(name, vars); err != nil {
			return err
		}
		if token != "" {
			if err := mgr.SaveToken(name, token); err != nil {
				return err
			}
		}
		fmt.Printf("Added profile: %s\n", name)
		return nil
	case "edit":
		if len(args) < 2 {
			return errors.New("usage: anthro-env edit <name>")
		}
		name := args[1]
		vars, err := mgr.ReadProfile(name)
		if err != nil {
			return fmt.Errorf("profile not found: %s", name)
		}
		reader := bufio.NewReader(os.Stdin)

		currentBaseURL := strings.TrimSpace(vars["ANTHROPIC_BASE_URL"])
		fmt.Printf("ANTHROPIC_BASE_URL [keep: %s]: ", core.OrDefault(currentBaseURL, "<empty>"))
		baseURL, err := readInputLine(reader)
		if err != nil {
			return err
		}
		if baseURL != "" {
			vars["ANTHROPIC_BASE_URL"] = baseURL
		}

		currentModel := strings.TrimSpace(vars["ANTHROPIC_MODEL"])
		if currentModel == "" {
			currentModel = strings.TrimSpace(vars["ANTHROPIC_SMALL_FAST_MODEL"])
		}
		fmt.Printf("ANTHROPIC_MODEL [keep: %s, '-' to clear]: ", core.OrDefault(currentModel, "<empty>"))
		model, err := readInputLine(reader)
		if err != nil {
			return err
		}
		switch model {
		case "":
			// keep
		case "-":
			delete(vars, "ANTHROPIC_MODEL")
			delete(vars, "ANTHROPIC_SMALL_FAST_MODEL")
			delete(vars, "ANTHROPIC_DEFAULT_SONNET_MODEL")
			delete(vars, "ANTHROPIC_DEFAULT_OPUS_MODEL")
			delete(vars, "ANTHROPIC_DEFAULT_HAIKU_MODEL")
		default:
			vars["ANTHROPIC_MODEL"] = model
			vars["ANTHROPIC_SMALL_FAST_MODEL"] = model
			vars["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
			vars["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
			vars["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
		}

		editTokenHint := "leave empty to keep, '-' to clear"
		if core.IsSSHSession() {
			editTokenHint = "SSH mode: leave empty to keep, '-' to clear, stored in plaintext"
		}
		fmt.Printf("API credential [%s, exported as ANTHROPIC_API_KEY for MiniMax and ANTHROPIC_AUTH_TOKEN otherwise]: ", editTokenHint)
		token, err := readInputLine(reader)
		if err != nil {
			return err
		}

		if token == "-" {
			if err := mgr.DeleteToken(name); err != nil {
				return err
			}
			token = ""
		}

		if err := mgr.SaveProfileAndToken(name, vars, token); err != nil {
			return err
		}
		fmt.Printf("Updated profile: %s\n", name)
		return nil
	case "rm":
		if len(args) < 2 {
			return errors.New("usage: anthro-env rm <name>")
		}
		if err := mgr.RemoveProfile(args[1]); err != nil {
			return err
		}
		fmt.Printf("Removed profile: %s\n", args[1])
		return nil
	default:
		return fmt.Errorf("unknown profile command: %s", args[0])
	}
}

func runDoctor(mgr *core.Manager) error {
	reports := mgr.Doctor()
	for _, r := range reports {
		fmt.Printf("[%s] %s\n", r.Status, r.Message)
	}
	return nil
}

func runMigrateTokens(mgr *core.Manager) error {
	migrated, skipped, err := mgr.MigratePlaintextTokens()
	if err != nil {
		return err
	}
	fmt.Printf("Token migration finished. migrated=%d skipped=%d\n", migrated, skipped)
	if migrated > 0 {
		fmt.Println("Plaintext API credentials have been removed from migrated profile files.")
	}
	return nil
}

func runExport(mgr *core.Manager) error {
	snippet, err := mgr.ExportSnippet()
	if err != nil {
		return err
	}
	fmt.Print(snippet)
	return nil
}

// readInputLine reads one line from stdin, trims spaces, and returns a non-nil error only on I/O failure (not on EOF).
func readInputLine(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	s = strings.TrimSpace(s)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return s, nil
		}
		return "", err
	}
	return s, nil
}

func printUsage() {
	fmt.Println(`anthro-env commands:
  anthro-env -v | --version
  anthro-env migrate-tokens
  anthro-env init
  anthro-env menu
  anthro-env add <name>
  anthro-env edit <name>
  anthro-env use <name>
  anthro-env ls
  anthro-env current
  anthro-env rm <name>
  anthro-env hook <zsh|bash>
  anthro-env env | export
  anthro-env doctor`)
}
