@echo off
setlocal

:: remember original path
:: set old_cd="%cd%"
rmdir /q/s expected
mkdir expected
rmdir /q/s got
mkdir got

:: cd %1
for /r %1\testdata %%f in (*.txt.png) do (
	copy %%f expected\
)
for /r %1\tmp\testimgs %%f in (*.png) do (
	copy %%f got\
)

:: restore original path
:: cd /d %old_cd%

endlocal

