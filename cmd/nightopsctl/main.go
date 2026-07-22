// Command nightopsctl provides a small, scriptable client for a local NightOps instance.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jordanistan/nightops/internal/api"
	"github.com/jordanistan/nightops/internal/atlas"
	syncbundle "github.com/jordanistan/nightops/internal/sync"
)

func main() {
	baseURL := flag.String("addr", "http://127.0.0.1:8787", "NightOps API URL")
	authEnv := flag.String("auth-env", "", "environment variable containing the API authentication value")
	atlasVersion := flag.String("atlas-version", "community-local", "version assigned to an Atlas contribution")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}
	client := api.NewClient(*baseURL)
	if *authEnv != "" {
		client.AuthValue = os.Getenv(*authEnv)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var err error
	switch flag.Arg(0) {
	case "status":
		var value api.Status
		value, err = client.Status(ctx)
		err = printJSON(value, err)
	case "missions":
		var value []api.Mission
		value, err = client.Missions(ctx)
		err = printJSON(value, err)
	case "mission":
		var value api.Mission
		value, err = client.Mission(ctx, flag.Arg(1))
		err = printJSON(value, err)
	case "export-sync":
		err = exportSync(ctx, client, flag.Arg(1))
	case "import-sync":
		err = importSync(ctx, client, flag.Arg(1))
	case "atlas-validate":
		err = validateAtlas(flag.Arg(1), *atlasVersion)
	case "atlas-package":
		err = packageAtlas(flag.Arg(1), flag.Arg(2), *atlasVersion)
	default:
		err = fmt.Errorf("unknown command %q", flag.Arg(0))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "nightopsctl:", err)
		os.Exit(1)
	}
}

func exportSync(ctx context.Context, client *api.Client, path string) error {
	if path == "" {
		return fmt.Errorf("export-sync requires an output path")
	}
	bundle, err := client.ExportSync(ctx)
	if err != nil {
		return err
	}
	if err := syncbundle.Save(path, bundle); err != nil {
		return fmt.Errorf("save sync bundle: %w", err)
	}
	fmt.Printf("sync bundle exported to %s\n", path)
	return nil
}

func importSync(ctx context.Context, client *api.Client, path string) error {
	if path == "" {
		return fmt.Errorf("import-sync requires an input path")
	}
	bundle, err := syncbundle.Load(path)
	if err != nil {
		return err
	}
	report, err := client.ImportSync(ctx, bundle)
	if err != nil {
		return err
	}
	return printJSON(report, nil)
}

func validateAtlas(path, version string) error {
	catalog, err := atlas.LoadCSV(path, version)
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"valid": true, "version": catalog.Version, "locations": len(catalog.Locations)}, nil)
}

func packageAtlas(input, output, version string) error {
	if input == "" || output == "" {
		return fmt.Errorf("atlas-package requires input CSV and output JSON paths")
	}
	catalog, err := atlas.LoadCSV(input, version)
	if err != nil {
		return err
	}
	contribution, err := atlas.NewContributionPackage(catalog, time.Now().UTC())
	if err != nil {
		return err
	}
	if err := atlas.SaveContribution(output, contribution); err != nil {
		return err
	}
	fmt.Printf("Atlas contribution package written to %s\n", output)
	return nil
}

func printJSON(value any, err error) error {
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(value)
}
func usage() {
	fmt.Fprintln(os.Stderr, "usage: nightopsctl [flags] status|missions|mission ID|export-sync PATH|import-sync PATH|atlas-validate CSV|atlas-package CSV JSON")
}
