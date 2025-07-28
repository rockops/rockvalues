//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// ProcessInfo contient les informations d'un processus
type ProcessInfo struct {
	PID     int
	PPID    int
	Name    string
	CmdLine string
}

// GetParentProcesses retourne la chaîne des processus parents
func GetParentProcesses() ([]ProcessInfo, error) {
	var processes []ProcessInfo

	currentPID := os.Getpid()

	for currentPID > 0 {
		info, err := getProcessInfoUnix(currentPID)
		if err != nil {
			break
		}

		processes = append(processes, info)

		// Si c'est le processus init (PID 1) ou si PPID = PID, on s'arrête
		if info.PPID <= 1 || info.PPID == info.PID {
			break
		}

		currentPID = info.PPID
	}

	return processes, nil
}

func getProcessInfoUnix(pid int) (ProcessInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return getProcessInfoLinux(pid)
	case "darwin":
		return getProcessInfoDarwin(pid)
	default:
		return ProcessInfo{}, fmt.Errorf("OS non supporté: %s", runtime.GOOS)
	}
}

// getProcessInfoLinux lit les informations depuis /proc
func getProcessInfoLinux(pid int) (ProcessInfo, error) {
	info := ProcessInfo{PID: pid}

	// Lire /proc/PID/stat pour obtenir le PPID et le nom
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statFile, err := os.Open(statPath)
	if err != nil {
		return info, err
	}
	defer statFile.Close()

	scanner := bufio.NewScanner(statFile)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 {
			// Le PPID est le 4ème champ (index 3)
			if ppid, err := strconv.Atoi(fields[3]); err == nil {
				info.PPID = ppid
			}
			// Le nom est le 2ème champ (index 1), entouré de parenthèses
			if len(fields[1]) > 2 {
				info.Name = fields[1][1 : len(fields[1])-1]
			}
		}
	}

	// Lire /proc/PID/cmdline pour obtenir la ligne de commande
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmdlineData, err := os.ReadFile(cmdlinePath)
	if err == nil {
		// Les arguments sont séparés par des caractères null
		cmdline := string(cmdlineData)
		cmdline = strings.ReplaceAll(cmdline, "\x00", " ")
		cmdline = strings.TrimSpace(cmdline)
		if cmdline == "" {
			cmdline = "[" + info.Name + "]"
		}
		info.CmdLine = cmdline
	} else {
		info.CmdLine = "[" + info.Name + "]"
	}

	return info, nil
}

// getProcessInfoDarwin utilise les informations système de macOS
func getProcessInfoDarwin(pid int) (ProcessInfo, error) {
	//info := ProcessInfo{PID: pid}

	// Sur macOS, nous devons utiliser les syscalls ou des commandes système
	// Pour une solution portable, nous utiliserons une approche simplifiée

	// Essayer de lire depuis /proc si disponible (rare sur macOS)
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err == nil {
		return getProcessInfoLinux(pid)
	}

	// Fallback: utiliser ps de manière limitée pour obtenir les infos de base
	// Note: Ceci est un compromis pour la portabilité
	return getProcessInfoFallback(pid)
}

func getProcessInfoFallback(pid int) (ProcessInfo, error) {
	// Cette fonction pourrait utiliser des syscalls spécifiques à l'OS
	// Pour l'instant, retournons des informations limitées
	return ProcessInfo{
		PID:     pid,
		PPID:    1, // Valeur par défaut
		Name:    "unknown",
		CmdLine: "unknown",
	}, nil
}

func GetHelmCmd() (ProcessInfo, error) {
	processes, err := GetParentProcesses()
	if err != nil {
		Fdebug("Erreur: %v\n", err)
		return ProcessInfo{}, err
	}

	helm := os.Getenv("HELM_BIN")
	if helm == "" {
		helm = "helm"
	}

	for _, proc := range processes {
		if proc.Name == helm {
			return proc, nil
		}
	}
	return ProcessInfo{}, nil
}
