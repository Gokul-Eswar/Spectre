; -- SPECTRE Inno Setup Script --
; Produces a single, self-contained Windows Installer (Spectre-Setup-x64.exe)

#ifndef MyAppVersion
#define MyAppVersion "1.0.0"
#endif

#define MyAppName "SPECTRE"
#define MyAppPublisher "SPECTRE Project"
#define MyAppURL "https://github.com/Gokul-Eswar/Spectre"
#define MyAppExeName "spectre.exe"

[Setup]
AppId={{8B1D4F42-99D7-4C3A-A2E1-692762BD0E18}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
AllowNoIcons=yes
OutputDir=..\..\dist
OutputBaseFilename=Spectre-Setup-x64
Compression=lzma2/ultra64
SolidCompression=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
ChangesEnvironment=yes
UninstallDisplayIcon={app}\{#MyAppExeName}
DisableProgramGroupPage=yes
CloseApplications=force

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "addtopath"; Description: "Add SPECTRE to System PATH (accessible from any terminal)"; GroupDescription: "System Integration:"; Flags: checkedonce
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "..\..\dist\bundle\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\{#MyAppName} CLI"; Filename: "{cmd}"; Parameters: "/K """"{app}\{#MyAppExeName}"""" --help"; WorkingDir: "{app}"
Name: "{group}\{#MyAppName} Interactive Console"; Filename: "{cmd}"; Parameters: "/K """"{app}\{#MyAppExeName}"""" console"; WorkingDir: "{app}"
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{cmd}"; Parameters: "/K """"{app}\{#MyAppExeName}"""" --help"; WorkingDir: "{app}"; Tasks: desktopicon

[Code]
const
  EnvironmentKey = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment';

procedure CurStepChanged(CurStep: TSetupStep);
var
  Path: string;
  AppDir: string;
  UpperAppDir: string;
  UpperPath: string;
begin
  if (CurStep = ssPostInstall) and IsTaskSelected('addtopath') then
  begin
    AppDir := ExpandConstant('{app}');
    UpperAppDir := UpperCase(AppDir);

    if RegQueryStringValue(HKEY_LOCAL_MACHINE, EnvironmentKey, 'Path', Path) then
    begin
      UpperPath := ';' + UpperCase(Path) + ';';
      if Pos(';' + UpperAppDir + ';', UpperPath) = 0 then
      begin
        if (Length(Path) > 0) and (Path[Length(Path)] <> ';') then
          Path := Path + ';';
        Path := Path + AppDir;
        RegWriteExpandStringValue(HKEY_LOCAL_MACHINE, EnvironmentKey, 'Path', Path);
      end;
    end;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Path: string;
  AppDir: string;
  UpperAppDir: string;
  UpperPath: string;
  P: Integer;
begin
  if CurUninstallStep = usUninstall then
  begin
    AppDir := ExpandConstant('{app}');
    UpperAppDir := UpperCase(AppDir);

    if RegQueryStringValue(HKEY_LOCAL_MACHINE, EnvironmentKey, 'Path', Path) then
    begin
      UpperPath := ';' + UpperCase(Path) + ';';
      P := Pos(';' + UpperAppDir + ';', UpperPath);
      if P > 0 then
      begin
        if P = 1 then
          Delete(Path, 1, Length(AppDir) + 1)
        else
          Delete(Path, P, Length(AppDir) + 1);
        RegWriteExpandStringValue(HKEY_LOCAL_MACHINE, EnvironmentKey, 'Path', Path);
      end;
    end;
  end;
end;
