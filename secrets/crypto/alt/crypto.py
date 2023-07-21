from cryptography.hazmat.primitives.serialization import load_ssh_private_key

def key(path: str):
    '''Load private key from filesystem'''
    with open(path, 'rb') as f:
        return load_ssh_private_key(f.read(), password=None)
