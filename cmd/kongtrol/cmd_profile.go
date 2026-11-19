package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
)

// ── profile ───────────────────────────────────────────────────────────────────

var profileCmd = &cobra.Command{
	Use: "profile",
}

var (
	profileImportName      string
	profileImportOverwrite bool
)

// profile import: the CLI side of the dashboard's "Importar archivo"
// (POST /api/v1/vpns/import). Same checks and storage
// (internal/profileimport): the file is validated like the privileged
// helper does and stored 0600 in ~/.kongtrol/configs/. It needs no
// kongtrol.yaml and no running daemon.
var profileImportCmd = &cobra.Command{
	Use:         "import <file>",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{annotationPrefsOnly: "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
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
		fmt.Println(tuiInfo(cf("cli.profile.import.next", res.Type, res.Path)))
		return nil
	},
}

// runProfileImport reads file and imports it under name (default: the
// file name without its extension); the type comes from the extension.
func runProfileImport(file, name string, overwrite bool) (profileimport.Result, error) {
	typ := profileimport.TypeForFile(file)
	if typ == "" {
		return profileimport.Result{}, fmt.Errorf("%s", cf("cli.profile.import.ext", filepath.Base(file)))
	}
	fi, err := os.Stat(file)
	if err != nil || !fi.Mode().IsRegular() {
		return profileimport.Result{}, fmt.Errorf("%s", cf("cli.profile.import.read", file))
	}
	if fi.Size() > profileimport.MaxSize {
		return profileimport.Result{}, localizeConfigErr(profileimport.TooBig())
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return profileimport.Result{}, fmt.Errorf("%s", cf("cli.profile.import.read", file))
	}
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	}
	dir, err := profileimport.DefaultDir()
	if err != nil {
		return profileimport.Result{}, err
	}
	res, err := profileimport.Import(dir, profileimport.Request{Name: name, Type: typ, Content: content, Overwrite: overwrite})
	if errors.Is(err, profileimport.ErrStore) {
		return res, fmt.Errorf("%s\n    %s", cf("cli.profile.import.store", dir), err)
	}
	return res, localizeConfigErr(err)
}

func init() {
	profileImportCmd.Flags().StringVar(&profileImportName, "name", "", ct("cli.profile.import.flag.name"))
	profileImportCmd.Flags().BoolVar(&profileImportOverwrite, "overwrite", false, ct("cli.profile.import.flag.overwrite"))
	profileCmd.AddCommand(profileImportCmd)
	rootCmd.AddCommand(profileCmd)
}
