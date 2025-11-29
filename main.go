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
	"strings"
)

const (
	moduleCertd = "certd"
	moduleAcme  = "acme.sh"
)

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

	flag.StringVar(&opt.module, "module", "", "Module preset for certificate paths (certd|acme.sh)")
	flag.StringVar(&opt.module, "m", "", "Module preset for certificate paths (certd|acme.sh)")
	flag.StringVar(&opt.subscription, "subscription", "", "Plesk subscription name or 'admin' for admin pool (required)")
	flag.StringVar(&opt.subscription, "s", "", "Plesk subscription name or 'admin' for admin pool (required)")
	flag.StringVar(&opt.name, "name", "", "Certificate name (required)")
	flag.StringVar(&opt.name, "n", "", "Certificate name (required)")
	flag.StringVar(&opt.pubPath, "pub", "", "Public certificate file path")
	flag.StringVar(&opt.priPath, "pri", "", "Private key file path")
	flag.StringVar(&opt.caPath, "ca", "", "CA certificate file path (optional)")

	flag.Parse()

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
	if opt.module != "" && opt.module != moduleCertd && opt.module != moduleAcme {
		return opt, fmt.Errorf("unsupported module '%s' (allowed: certd, acme.sh)", opt.module)
	}

	return opt, nil
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
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, name) {
			return true
		}
	}
	return false
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

	deployOutput, err := deployCertificate(opt.subscription, opt.name, certPath, keyPath, caPath, exists)
	if err != nil {
		fmt.Fprint(os.Stderr, deployOutput)
		action := "create"
		if exists {
			action = "update"
		}
		log.Fatalf("Failed to %s certificate: %v", action, err)
	}

	fmt.Printf("Certificate '%s' deployed successfully to subscription '%s'.\n", opt.name, opt.subscription)
}
