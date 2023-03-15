Name "ssh-sync Installer"
OutFile "ssh-sync-installer.exe"
InstallDir "$LOCALAPPDATA\ssh-sync"

RequestExecutionLevel user

; InstallDirRegKey HKLM "Software\ssh-sync" "Install_Dir"

Section "ssh-sync (required)"
 SectionIn RO
  
  ; Set output path to the installation directory.
  SetOutPath $INSTDIR
  
  ; Put file there (you can add more File lines too)
  File "ssh-sync.exe"
  ; Wildcards are allowed:
  ; File *.dll
  ; To add a folder named MYFOLDER and all files in it recursively, use this EXACT syntax:
  ; File /r MYFOLDER\*.*
  ; See: https://nsis.sourceforge.io/Reference/File
  ; MAKE SURE YOU PUT ALL THE FILES HERE IN THE UNINSTALLER TOO
  
  ; Write the installation path into the registry
  ; WriteRegStr HKLM SOFTWARE\ssh-sync "Install_Dir" "$INSTDIR"
  
  ; Write the uninstall keys for Windows
  ; WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ssh-sync" "DisplayName" "ssh-sync"
  ; WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ssh-sync" "UninstallString" '"$INSTDIR\uninstall.exe"'
  ; WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ssh-sync" "NoModify" 1
  ; WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ssh-sync" "NoRepair" 1
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Post Installation"
  EnVar::Check "PATH" "$INSTDIR"
  Pop $0
    StrCmp $0 "0" +2
    EnVar::AddValue "PATH" "$INSTDIR"
SectionEnd

Section "Uninstall"
  
  ; Remove registry keys
  ; DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ssh-sync"
  ; DeleteRegKey HKLM SOFTWARE\ssh-sync

  ; Remove files and uninstaller
  ; MAKE SURE NOT TO USE A WILDCARD. IF A
  ; USER CHOOSES A STUPID INSTALL DIRECTORY,
  ; YOU'LL WIPE OUT OTHER FILES TOO
  Delete $INSTDIR\ssh-sync.exe
  Delete $INSTDIR\uninstall.exe

  ; Remove shortcuts, if any
  ; Delete "$SMPROGRAMS\ssh-sync\*.*"

  ; Remove directories used (only deletes empty dirs)
  RMDir "$SMPROGRAMS\ssh-sync"
  RMDir "$INSTDIR"

  ; Remove from path
  EnVar::DeleteValue "PATH" "$INSTDIR"
SectionEnd
