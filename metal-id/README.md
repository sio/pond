# Machine identity based on hardware configuration

This tool calculates a hardware fingerprint based on information exposed to
Linux system and derives a deterministic SSH key pair from it. That key pair
can be used for machine-to-machine authentication.

## Security

Although author is not aware of any successful attack vectors you should keep
in mind that this tool was created by a hobbyist for personal use.

Deterministic key pair is no doubt less secure than a truly random (or even a
pseudo random) one. Author recommends to use only `ssh-keygen` to create keys
for anything you care about.
