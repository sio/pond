import json
import random
import sys
from base64 import b64encode

import coolname
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat

import v1
import crypto

def main():
    keypath = sys.argv[1]
    rounds = 10
    signer = crypto.key(keypath)
    output = {
        "key": signer.public_key().public_bytes(Encoding.OpenSSH, PublicFormat.OpenSSH).decode('utf-8'),
        "samples": [],
    }
    for _ in range(rounds):
        message = ';'.join(coolname.generate_slug() for _ in range(random.randint(1,9)))
        keywords = coolname.generate(random.randint(2, 4))
        signature, sig_nonce = v1._sign(signer, keywords)
        key, kdf_nonce = v1._kdf(signature)
        encrypted = v1.encrypt(signer, message, keywords)
        output["samples"].append(dict(
            message=message,
            signature_nonce=b64(sig_nonce),
            signature=b64(signature),
            kdf_nonce=b64(kdf_nonce),
            key=b64(key),
            encrypted=b64(encrypted),
        ))
    print(json.dumps(output, indent=2, ensure_ascii=False, sort_keys=True))

def b64(data: bytes) -> str:
    return b64encode(data).decode('utf-8')

if __name__ == '__main__':
    main()
