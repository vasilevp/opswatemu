# opswatemu
An OPSWAT emulator
## First-time setup
1. Run actual OPSWAT on a VM
2. Get that VM's HWID and save it for later use:
```bash
sudo dmidecode -s system-uuid | tr -d "-"
```
3. Get the emulator:
```bash
go get github.com/vasilevp/opswatemu
```
4. Plug the HWID in and run the emulator:
```bash
opswatemu --hwid ABCDEF...
```
or
```bash
HWID=ABCDEF... opswatemu
```
5. A status webpage will open - this is done to provide an easy way to add the self-signed certificate to browser exceptions.

After this initial setup you should be fine by just doing step 4 again.
