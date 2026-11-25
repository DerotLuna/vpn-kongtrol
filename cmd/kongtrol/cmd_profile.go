package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
)

// ── profile ───────────────────────────────────────────────────────────────────

var profileCmd = &cobra.Command{
	Use: "profile",
}

var (
	profileImportName      string
	profileImportOverwrite bool
	// profileImportExtra: certificates and keys given after the file
	// (ADR-0105).
	profileImportExtra []string
)

// profile import: the CLI side of the dashboard's "Importar archivo"
// (POST /api/v1/vpns/import). Same checks and storage
// (internal/profileimport): the file is validated like the privileged
// helper does and stored 0600 in ~/.kongtrol/configs/. It needs no
// kongtrol.yaml and no running daemon.
var profileImportCmd = &cobra.Command{
	Use:         "import <file> [certificates...]",
	Args:        cobra.RangeArgs(1, 1+profileimport.MaxExtraFiles),
	Annotations: map[string]string{annotationPrefsOnly: "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		profileImportExtra = args[1:]
		if profileImportCreate {
			return importAndCreate(args[0], profileImportName, profileImportOverwrite)
		}
		res, err := runProfileImport(args[0], profileImportName, profileImportOverwrite)
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(res)
		}
		key := "cli.profile.import.done"
		if res.Overwritten {
			key = "cli.profile.import.replaced"
		}
		fmt.Println(tuiOK(cf(key, res.Path)))
		printIncluded(res)
		fmt.Println(tuiInfo(cf("cli.profile.import.next", res.Type, res.Path)))
		printImportHint(res)
		return nil
	},
}

// runProfileImport reads file and imports it under name (default: the
// file name without its extension); the type comes from the extension and,
// for a .conf, from its content (OpenVPN GUI's .conf files, ADR-0080).
// A file only root can read (e.g. /etc/wireguard/wg0.conf) is read with
// one permission prompt after asking (importRootOnly, ADR-0058).
func runProfileImport(file, name string, overwrite bool) (profileimport.Result, error) {
	typ := profileimport.TypeForFile(file)
	if typ == "" {
		return profileimport.Result{}, fmt.Errorf("%s", cf("cli.profile.import.ext", filepath.Base(file)))
	}
	fi, err := os.Stat(file)
	if err != nil && config.IsPermission(err) {
		return importRootOnly(absPath(file), name, overwrite, profileImportYes)
	}
	if err != nil || !fi.Mode().IsRegular() {
		return profileimport.Result{}, fmt.Errorf("%s", cf("cli.profile.import.read", file))
	}
	if fi.Size() > profileimport.MaxSize {
		return profileimport.Result{}, localizeConfigErr(profileimport.TooBig())
	}
	content, err := os.ReadFile(file)
	if err != nil && config.IsPermission(err) {
		return importRootOnly(absPath(file), name, overwrite, profileImportYes)
	}
	if err != nil {
		return profileimport.Result{}, fmt.Errorf("%s", cf("cli.profile.import.read", file))
	}
	refs, err := importRefs(file, profileImportExtra)
	if err != nil {
		return profileimport.Result{}, err
	}
	return storeImport(file, name, profileimport.TypeForContent(file, content), content, overwrite, refs)
}

// absPath is p made absolute (the elevated read takes absolute paths only).
func absPath(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

// importRefs: the files an .ovpn names are read from its own folder, and
// from the extra files given after it (matched by base name, ADR-0105); an
// absolute path (or a "../" leaving the folder) is read too, with the
// stricter outside checks (ADR-0112). The elevated root-only read passes no
// refs at all.
func importRefs(file string, extra []string) (*profileimport.Refs, error) {
	files := make(map[string][]byte, len(extra))
	for _, p := range extra {
		fi, err := os.Stat(p)
		if err != nil || !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%s", cf("cli.profile.import.extra_read", p))
		}
		if fi.Size() > profileimport.MaxReferenceSize {
			return nil, localizeConfigErr(&config.ValidationError{Key: profileimport.ErrKeyRefTooBig, Args: []any{filepath.Base(p), profileimport.MaxReferenceSize >> 10}})
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("%s", cf("cli.profile.import.extra_read", p))
		}
		files[filepath.Base(p)] = b
	}
	refs, err := profileimport.NewRefs(filepath.Dir(absPath(file)), files)
	if err != nil {
		return nil, localizeConfigErr(err)
	}
	refs.Disk = true
	return refs, nil
}

// storeImport validates content and stores it in ~/.kongtrol/configs.
func storeImport(file, name, typ string, content []byte, overwrite bool, refs *profileimport.Refs) (profileimport.Result, error) {
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	}
	dir, err := profileimport.DefaultDir()
	if err != nil {
		return profileimport.Result{}, err
	}
	res, err := profileimport.Import(dir, profileimport.Request{Name: name, Type: typ, Content: content, Overwrite: overwrite, Refs: refs})
	if errors.Is(err, profileimport.ErrStore) {
		return res, fmt.Errorf("%s\n    %s", cf("cli.profile.import.store", dir), err)
	}
	var miss *profileimport.MissingFilesError
	if errors.As(err, &miss) {
		// The panel's "Agregar" is the extra arguments here.
		return res, fmt.Errorf("%s\n    %s", localizeConfigErr(err), cf("cli.profile.import.missing_hint", file, strings.Join(miss.Names, " ")))
	}
	return res, localizeConfigErr(err)
}

// printIncluded says which referenced files went inside the stored profile.
func printIncluded(res profileimport.Result) {
	if len(res.Included) > 0 {
		fmt.Println(tuiInfo(cf("cli.profile.import.included", strings.Join(res.Included, ", "))))
	}
}

func init() {
	profileImportCmd.Flags().StringVar(&profileImportName, "name", "", ct("cli.profile.import.flag.name"))
	profileImportCmd.Flags().BoolVar(&profileImportOverwrite, "overwrite", false, ct("cli.profile.import.flag.overwrite"))
	profileCmd.AddCommand(profileImportCmd)
	rootCmd.AddCommand(profileCmd)
}

// printImportHint prints the line for a file from a provider account
// (Proton: do not connect it while Proton's app is connected, ADR-0103).
func printImportHint(res profileimport.Result) {
	if key := profileimport.HintKey(res.Provider); key != "" {
		fmt.Println(tuiWarn(ct(key)))
	}
}
