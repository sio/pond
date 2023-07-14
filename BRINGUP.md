# Infrastructure Bringup Sequence

## Boot

- From PXE server (if such server is already available in LAN)
- From an USB stick (first machine in LAN will assume router duties)
- From NIC ROM (if I ever get around to flashing iPXE there)

## iPXE

- Fetches boot menu from HTTPS (no auth)
- Fetches kernel+initramfs from HTTPS (no auth)
- Boots into initramfs

## Initramfs system

- Generates unique hardware derived authentication key
- Fetches confidential values from secrets storage using that key
- Decides which rootfs revision to boot into based on iPXE menu input
  and on confidential values fetched in previous step
- Decides what medium to use for rootfs (in order of decreasing priority):
    - Network storage
    - Local scratch volume
    - Ramdisk
- Prepares rootfs for mounting (populates ramdisk)
- Launches main init (systemd)

## Linux hypervisor

- Uses configuration management (cloud-init? Ansible?) to finish OS
  customization. Secrets are fetched from remote storage using the same
  hardware derived key.
- Launches virtual machines as specified by CM tool
- Passes required devices to VMs

## Virtual machines

- All productive work happens within VMs: router, nas, k8s masters/workers,
  development playground
- VMs "own" required hardware: NAS owns HDDs, print server owns USB printer,
  router owns NIC
- VM disks are stored on NAS
- VMs are created from Debian cloud images with one-time CM tool (cloud-init)
- Scheduled VM maintenance: upgrades, patching **=???**
