package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	moduleCertd     = "certd"
	moduleAcme      = "acme.sh"
	moduleCertimate = "certimate"
)

type flagInfo struct {
	long  string
	short string
	usage string
}

type options struct {
	module       string
	subscription string
	name         string
	pubPath      string
	priPath      string
	caPath       string
}

func parseFlags() (options, error) {
	var opt options

	flagSet := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)
	flagSet.StringVar(&opt.module, "module", "", "Module preset for certificate paths (certd|acme.sh|certimate)")
	flagSet.StringVar(&opt.subscription, "subscription", "", "Plesk subscription name or 'admin' for admin pool (required)")
	flagSet.StringVar(&opt.name, "name", "", "Certificate name (required)")
	flagSet.StringVar(&opt.pubPath, "pub", "", "Public certificate file path")
	flagSet.StringVar(&opt.priPath, "pri", "", "Private key file path")
	flagSet.StringVar(&opt.caPath, "ca", "", "CA certificate file path (optional)")

	flagSet.Usage = func() {
		flags := []flagInfo{
			{long: "subscription", short: "s", usage: "Plesk subscription name or 'admin' for admin pool (required)"},
			{long: "name", short: "n", usage: "Certificate name (required)"},
			{long: "module", short: "m", usage: "Module preset for certificate paths (certd|acme.sh|certimate)"},
			{long: "pub", usage: "Public certificate file path"},
			{long: "pri", usage: "Private key file path"},
			{long: "ca", usage: "CA certificate file path (optional)"},
		}

		fmt.Fprintf(flagSet.Output(), "Usage of %s:\n", flagSet.Name())
		for _, f := range flags {
			if f.short != "" {
				fmt.Fprintf(flagSet.Output(), "  --%s, -%s string\n    \t%s\n", f.long, f.short, f.usage)
				continue
			}
			fmt.Fprintf(flagSet.Output(), "  --%s string\n    \t%s\n", f.long, f.usage)
		}
	}

	if err := flagSet.Parse(normalizeArgs(os.Args[1:])); err != nil {
		return opt, err
	}

	opt.module = strings.TrimSpace(opt.module)
	opt.subscription = strings.TrimSpace(opt.subscription)
	opt.name = strings.TrimSpace(opt.name)
	opt.pubPath = strings.TrimSpace(opt.pubPath)
	opt.priPath = strings.TrimSpace(opt.priPath)
	opt.caPath = strings.TrimSpace(opt.caPath)

	if opt.subscription == "" {
		return opt, errors.New("missing required --subscription/-s")
	}
	if opt.name == "" {
		return opt, errors.New("missing required --name/-n")
	}
	if opt.module != "" && opt.module != moduleCertd && opt.module != moduleAcme && opt.module != moduleCertimate {
		return opt, fmt.Errorf("unsupported module '%s' (allowed: certd, acme.sh, certimate)", opt.module)
	}

	return opt, nil
}

func normalizeArgs(args []string) []string {
	aliases := map[string]string{
		"-s": "--subscription",
		"-n": "--name",
		"-m": "--module",
	}

	var out []string
	for _, arg := range args {
		if replacement, ok := aliases[arg]; ok {
			out = append(out, replacement)
			continue
		}

		replaced := false
		for short, long := range aliases {
			prefix := short + "="
			if strings.HasPrefix(arg, prefix) {
				out = append(out, long+arg[len(short):])
				replaced = true
				break
			}
		}
		if replaced {
			continue
		}

		out = append(out, arg)
	}

	return out
}

func resolvePaths(opt options) (certPath, keyPath, caPath string, err error) {
	certPath = opt.pubPath
	keyPath = opt.priPath
	caPath = opt.caPath

	switch opt.module {
	case "":
	case moduleCertd:
		if certPath == "" {
			certPath = os.Getenv("HOST_CRT_PATH")
		}
		if keyPath == "" {
			keyPath = os.Getenv("HOST_KEY_PATH")
		}
		if caPath == "" {
			caPath = os.Getenv("HOST_IC_PATH")
		}
	case moduleAcme:
		if certPath == "" {
			certPath = os.Getenv("CERT_FULLCHAIN_PATH")
		}
		if keyPath == "" {
			keyPath = os.Getenv("CERT_KEY_PATH")
		}
		if caPath == "" {
			caPath = os.Getenv("CA_CERT_PATH")
		}
	case moduleCertimate:
		if certPath == "" {
			certPath = os.Getenv("CERTIMATE_DEPLOYER_CMDVAR_CERTIFICATE_PATH")
		}
		if keyPath == "" {
			keyPath = os.Getenv("CERTIMATE_DEPLOYER_CMDVAR_PRIVATEKEY_PATH")
		}
		if caPath == "" {
			caPath = os.Getenv("CERTIMATE_DEPLOYER_CMDVAR_CERTIFICATE_INTERMEDIA_PATH")
		}
	default:
		return "", "", "", fmt.Errorf("unsupported module '%s'", opt.module)
	}

	certPath = strings.TrimSpace(certPath)
	keyPath = strings.TrimSpace(keyPath)
	caPath = strings.TrimSpace(caPath)

	if certPath == "" || keyPath == "" {
		return "", "", "", errors.New("public and private key paths must be provided via flags or module environment variables")
	}

	return certPath, keyPath, caPath, nil
}

func ensureReadable(path, label string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%s path is empty", label)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s path: %w", label, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("%s file check failed: %w", label, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s path points to a directory, not a file", label)
	}

	f, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("%s file is not readable: %w", label, err)
	}
	_ = f.Close()

	return absPath, nil
}

func listCertificates(subscription string) (string, error) {
	args := []string{"bin", "certificate", "-l"}
	if subscription == "admin" {
		args = append(args, "-admin")
	} else {
		args = append(args, "-domain", subscription)
	}

	return runPleskCommand(args)
}

func certificateExists(output, name string) bool {
	for _, certName := range certificateNames(output) {
		if certName == name {
			return true
		}
	}
	return false
}

func certificateNames(output string) []string {
	var names []string
	for _, line := range strings.Split(output, "\n") {
		if name, ok := parseCertificateName(line); ok {
			names = append(names, name)
		}
	}
	return names
}

func parseCertificateName(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}

	fields := strings.Fields(line)
	if len(fields) < 6 {
		return "", false
	}

	if _, err := strconv.Atoi(fields[len(fields)-1]); err != nil {
		return "", false
	}

	nameParts := fields[4 : len(fields)-1]
	if len(nameParts) == 0 {
		return "", false
	}

	return strings.Join(nameParts, " "), true
}

func deployCertificate(subscription, name, certPath, keyPath, caPath string, exists bool) (string, error) {
	args := []string{"bin", "certificate"}
	if exists {
		args = append(args, "-u", name)
	} else {
		args = append(args, "-c", name)
	}

	if subscription == "admin" {
		args = append(args, "-admin")
	} else {
		args = append(args, "-domain", subscription)
	}

	args = append(args, "-key-file", keyPath, "-cert-file", certPath)
	if caPath != "" {
		args = append(args, "-cacert-file", caPath)
	}

	return runPleskCommand(args)
}

func isCertificateMissingError(output string) bool {
	return strings.Contains(output, "Unable to update certificate") && strings.Contains(output, "Certificate does not exist.")
}

func runPleskCommand(args []string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("plesk", args...)
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	return out.String(), err
}

func main() {
	log.SetFlags(0)

	opt, err := parseFlags()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	certPath, keyPath, caPath, err := resolvePaths(opt)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	certPath, err = ensureReadable(certPath, "certificate")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	keyPath, err = ensureReadable(keyPath, "private key")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	if caPath != "" {
		if caPath, err = ensureReadable(caPath, "CA certificate"); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	listOutput, err := listCertificates(opt.subscription)
	if err != nil {
		fmt.Fprint(os.Stderr, listOutput)
		log.Fatalf("Failed to list certificates: %v", err)
	}

	exists := certificateExists(listOutput, opt.name)
	action := "create"
	if exists {
		action = "update"
	}

	deployOutput, err := deployCertificate(opt.subscription, opt.name, certPath, keyPath, caPath, exists)
	if err != nil && exists && isCertificateMissingError(deployOutput) {
		action = "create"
		deployOutput, err = deployCertificate(opt.subscription, opt.name, certPath, keyPath, caPath, false)
	}

	if err != nil {
		fmt.Fprint(os.Stderr, deployOutput)
		log.Fatalf("Failed to %s certificate: %v", action, err)
	}

	fmt.Printf("Certificate '%s' deployed successfully to subscription '%s'.\n", opt.name, opt.subscription)
}
