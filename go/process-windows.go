//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// ProcessInfo contient les informations d'un processus
type ProcessInfo struct {
	PID     int
	PPID    int
	Name    string
	CmdLine string
}

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")

	// Kernel32 functions
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procGetCurrentProcess        = kernel32.NewProc("GetCurrentProcess")

	// Ntdll functions
	procNtQueryInformationProcess = ntdll.NewProc("NtQueryInformationProcess")
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	MAX_PATH           = 260

	// Process access rights
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010

	// Process information classes
	ProcessBasicInformation       = 0
	ProcessCommandLineInformation = 60
)

type PROCESSENTRY32 struct {
	Size              uint32
	Usage             uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	Threads           uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

type PROCESS_BASIC_INFORMATION struct {
	Reserved1       uintptr
	PebBaseAddress  uintptr
	Reserved2       [2]uintptr
	UniqueProcessId uintptr
	Reserved3       uintptr
}

type UNICODE_STRING struct {
	Length        uint16
	MaximumLength uint16
	Buffer        uintptr
}

// GetProcessCommandLine récupère la ligne de commande complète d'un processus
func GetProcessCommandLine(pid int) (string, error) {
	// Ouvrir le processus avec les droits nécessaires
	handle, _, err := procOpenProcess.Call(
		PROCESS_QUERY_INFORMATION|PROCESS_VM_READ,
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return "", fmt.Errorf("impossible d'ouvrir le processus %d: %v", pid, err)
	}
	defer procCloseHandle.Call(handle)

	// Méthode 1: Essayer d'utiliser ProcessCommandLineInformation (Windows Vista+)
	cmdline, err := getCommandLineViaProcessInfo(handle)
	if err == nil && cmdline != "" {
		return cmdline, nil
	}

	// Méthode 2: Lire depuis le PEB (Process Environment Block)
	cmdline, err = getCommandLineViaPEB(handle)
	if err == nil && cmdline != "" {
		return cmdline, nil
	}

	// Méthode 3: Fallback - utiliser le nom du processus
	info, err := getBasicProcessInfo(pid)
	if err == nil {
		return info.Name, nil
	}

	return "", fmt.Errorf("impossible de récupérer la ligne de commande pour le PID %d", pid)
}

// getCommandLineViaProcessInfo utilise NtQueryInformationProcess avec ProcessCommandLineInformation
func getCommandLineViaProcessInfo(handle uintptr) (string, error) {
	var unicodeString UNICODE_STRING
	var returnLength uint32

	// Première appel pour obtenir la taille nécessaire
	ret, _, _ := procNtQueryInformationProcess.Call(
		handle,
		ProcessCommandLineInformation,
		uintptr(unsafe.Pointer(&unicodeString)),
		unsafe.Sizeof(unicodeString),
		uintptr(unsafe.Pointer(&returnLength)),
	)

	if ret != 0 && returnLength == 0 {
		return "", fmt.Errorf("échec de la première requête NtQueryInformationProcess")
	}

	// Allouer un buffer pour la chaîne Unicode
	if unicodeString.Length == 0 {
		return "", fmt.Errorf("longueur de commande nulle")
	}

	buffer := make([]uint16, unicodeString.Length/2+1)

	// Lire la mémoire du processus pour obtenir la chaîne
	err := readProcessMemory(handle, unicodeString.Buffer, uintptr(unsafe.Pointer(&buffer[0])), uintptr(unicodeString.Length))
	if err != nil {
		return "", err
	}

	return syscall.UTF16ToString(buffer), nil
}

// getCommandLineViaPEB lit la ligne de commande depuis le Process Environment Block
func getCommandLineViaPEB(handle uintptr) (string, error) {
	var pbi PROCESS_BASIC_INFORMATION
	var returnLength uint32

	// Obtenir les informations de base du processus
	ret, _, _ := procNtQueryInformationProcess.Call(
		handle,
		ProcessBasicInformation,
		uintptr(unsafe.Pointer(&pbi)),
		unsafe.Sizeof(pbi),
		uintptr(unsafe.Pointer(&returnLength)),
	)

	if ret != 0 {
		return "", fmt.Errorf("échec NtQueryInformationProcess pour ProcessBasicInformation")
	}

	if pbi.PebBaseAddress == 0 {
		return "", fmt.Errorf("adresse PEB nulle")
	}

	// Lire l'adresse des paramètres du processus depuis le PEB
	var processParameters uintptr
	err := readProcessMemory(handle, pbi.PebBaseAddress+0x20, uintptr(unsafe.Pointer(&processParameters)), unsafe.Sizeof(processParameters))
	if err != nil {
		return "", err
	}

	if processParameters == 0 {
		return "", fmt.Errorf("paramètres de processus nuls")
	}

	// Lire la structure UNICODE_STRING de la ligne de commande
	var cmdlineUnicodeString UNICODE_STRING
	err = readProcessMemory(handle, processParameters+0x70, uintptr(unsafe.Pointer(&cmdlineUnicodeString)), unsafe.Sizeof(cmdlineUnicodeString))
	if err != nil {
		return "", err
	}

	if cmdlineUnicodeString.Length == 0 || cmdlineUnicodeString.Buffer == 0 {
		return "", fmt.Errorf("ligne de commande vide")
	}

	// Lire la chaîne de caractères de la ligne de commande
	buffer := make([]uint16, cmdlineUnicodeString.Length/2+1)
	err = readProcessMemory(handle, cmdlineUnicodeString.Buffer, uintptr(unsafe.Pointer(&buffer[0])), uintptr(cmdlineUnicodeString.Length))
	if err != nil {
		return "", err
	}

	return syscall.UTF16ToString(buffer), nil
}

// readProcessMemory lit la mémoire d'un processus
func readProcessMemory(handle, address, buffer uintptr, size uintptr) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procReadProcessMemory := kernel32.NewProc("ReadProcessMemory")

	var bytesRead uintptr
	ret, _, err := procReadProcessMemory.Call(
		handle,
		address,
		buffer,
		size,
		uintptr(unsafe.Pointer(&bytesRead)),
	)

	if ret == 0 {
		return fmt.Errorf("ReadProcessMemory failed: %v", err)
	}

	return nil
}

// getBasicProcessInfo récupère les informations de base d'un processus
func getBasicProcessInfo(pid int) (ProcessInfo, error) {
	info := ProcessInfo{PID: pid}

	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == uintptr(syscall.InvalidHandle) {
		return info, fmt.Errorf("impossible de créer le snapshot")
	}
	defer procCloseHandle.Call(snapshot)

	var pe32 PROCESSENTRY32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
	if ret == 0 {
		return info, fmt.Errorf("aucun processus trouvé")
	}

	for {
		if pe32.ProcessID == uint32(pid) {
			info.PPID = int(pe32.ParentProcessID)
			info.Name = syscall.UTF16ToString(pe32.ExeFile[:])
			return info, nil
		}

		ret, _, _ := procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
		if ret == 0 {
			break
		}
	}

	return info, fmt.Errorf("processus %d non trouvé", pid)
}

// GetParentProcessesWithCommandLine récupère tous les processus parents avec leurs lignes de commande
func GetParentProcessesWithCommandLine() ([]ProcessInfo, error) {
	var processes []ProcessInfo
	currentPID := os.Getpid()

	for currentPID > 0 {
		// Obtenir les informations de base
		basicInfo, err := getBasicProcessInfo(currentPID)
		if err != nil {
			break
		}

		// Obtenir la ligne de commande complète
		cmdline, err := GetProcessCommandLine(currentPID)
		if err != nil {
			// Si on ne peut pas obtenir la ligne de commande, utiliser le nom
			cmdline = basicInfo.Name
		}

		processInfo := ProcessInfo{
			PID:     basicInfo.PID,
			PPID:    basicInfo.PPID,
			Name:    basicInfo.Name,
			CmdLine: cmdline,
		}

		processes = append(processes, processInfo)

		// Arrêter si on atteint le processus init ou un processus système
		if basicInfo.PPID <= 4 || basicInfo.PPID == basicInfo.PID {
			break
		}

		currentPID = basicInfo.PPID
	}

	return processes, nil
}

func GetHelmCmd() (ProcessInfo, error) {
	processes, err := GetParentProcessesWithCommandLine()
	if err != nil {
		Fdebug("Erreur: %v\n", err)
		return ProcessInfo{}, nil
	}

	helm := os.Getenv("HELM_BIN")
	if helm == "" {
		helm = "helm.exe"
	}

	helm = filepath.Base(helm)

	for _, proc := range processes {
		if proc.Name == helm {
			return proc, nil
		}
	}

	return ProcessInfo{}, nil
}
