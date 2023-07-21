import os
import nacl.secret
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.hkdf import HKDF

magic_header = b'pond/secret'
magic_separator = b'\0\n\r\n\0'

tag = b'\1'
ssh_nonce_bytes = 32
kdf_nonce_bytes = 32
box_nonce_bytes = 24
padding_max_bytes = 32
box_key_bytes = 32

def encrypt(signer, value: str, keywords) -> bytes:
    signature, ssh_nonce = _sign(signer, keywords)
    key, kdf_nonce = _kdf(signature)
    box_nonce = os.urandom(box_nonce_bytes)
    padding = os.urandom(padding_max_bytes)
    padding = padding[:1+padding[0]%padding_max_bytes]
    box = nacl.secret.SecretBox(key)
    return tag + ssh_nonce + kdf_nonce + box_nonce + box.encrypt(padding + value.encode('utf-8'))

def decrypt(signer, ciphertext: bytes, keywords) -> str:
    pass

def _sign(signer, keywords, nonce=None):
    if nonce is None:
        nonce = os.urandom(ssh_nonce_bytes)
    message = magic_separator.join([magic_header, nonce] + [s.encode('utf-8') for s in keywords])
    return signer.sign(message), nonce

def _kdf(signature, nonce=None):
    if nonce is None:
        nonce = os.urandom(kdf_nonce_bytes)
    hkdf = HKDF(
        algorithm=hashes.SHA256(),
        length=nonce[0] + signature[0] + box_key_bytes,
        salt=nonce,
        info=magic_header,
    )
    return hkdf.derive(signature)[nonce[0]+signature[0]:], nonce
