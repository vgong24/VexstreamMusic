//go:build windows

package lifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func discoverCandidatePorts() ([]sessionPort, error) {
	const script = `$ErrorActionPreference='Stop';$procs=@(Get-CimInstance Win32_Process | Where-Object {$_.Name -like 'VexStreamMusic*.exe'});$rows=@();foreach($p in $procs){$started='';if($p.CreationDate){$started=$p.CreationDate.ToString('yyyy-MM-ddTHH:mm:ssK')};$ls=@(Get-NetTCPConnection -State Listen -OwningProcess $p.ProcessId -ErrorAction SilentlyContinue);foreach($l in $ls){$rows += [PSCustomObject]@{pid=[int]$p.ProcessId;port=[int]$l.LocalPort;startedAt=$started}}};if($rows.Count -eq 0){'[]'}else{$rows | ConvertTo-Json -Compress}`
	command := exec.Command("powershell.exe", "-NoProfile", "-Command", script)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:     true,
		CreationFlags: 0x08000000,
	}
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("discover VexStream sessions: %w", err)
	}

	var rows []sessionPort
	raw := strings.TrimSpace(string(output))
	if raw == "" || raw == "null" {
		return []sessionPort{}, nil
	}
	if strings.HasPrefix(raw, "[") {
		var values []struct {
			PID       int    `json:"pid"`
			Port      int    `json:"port"`
			StartedAt string `json:"startedAt"`
		}
		if err := json.Unmarshal(output, &values); err != nil {
			return nil, fmt.Errorf("decode VexStream session scan: %w", err)
		}
		for _, value := range values {
			rows = append(rows, sessionPort{
				PID: value.PID, Port: value.Port, StartedAt: value.StartedAt,
			})
		}
		return rows, nil
	}

	var value struct {
		PID       int    `json:"pid"`
		Port      int    `json:"port"`
		StartedAt string `json:"startedAt"`
	}
	if err := json.Unmarshal(output, &value); err != nil {
		return nil, fmt.Errorf("decode VexStream session scan: %w", err)
	}
	return []sessionPort{{
		PID: value.PID, Port: value.Port, StartedAt: value.StartedAt,
	}}, nil
}

func stopSessionPID(pid int) error {
	if pid <= 0 || pid == os.Getpid() {
		return errors.New("refusing to stop the current VexStream process")
	}
	script := fmt.Sprintf(
		`$ErrorActionPreference='Stop';$p=Get-CimInstance Win32_Process -Filter "ProcessId=%d";if(-not $p){exit 0};if($p.Name -notlike 'VexStreamMusic*.exe'){throw 'PID no longer belongs to VexStreamMusic'};Stop-Process -Id %d -ErrorAction Stop`,
		pid,
		pid,
	)
	command := exec.Command("powershell.exe", "-NoProfile", "-Command", script)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:     true,
		CreationFlags: 0x08000000,
	}
	if err := command.Run(); err != nil {
		return fmt.Errorf("retire VexStream PID %d: %w", pid, err)
	}
	return nil
}
