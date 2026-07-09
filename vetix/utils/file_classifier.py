import os

# Refer to Openclaw upload whitelist
# https://github.com/openclaw/clawhub/blob/e8c3947b21175669352bd88ab8f7b00df624ee56/packages/clawdhub/src/schema/textFiles.ts
ALLOWED_EXT = {
    '.md', '.mdx', '.txt', '.json', '.json5', '.yaml', '.yml', '.toml',
    '.js', '.cjs', '.mjs', '.ts', '.tsx', '.jsx', '.py', '.sh', '.rb', '.go',
    '.rs', '.swift', '.kt', '.java', '.cs', '.cpp', '.c', '.h', '.hpp',
    '.sql', '.csv', '.ini', '.cfg', '.xml', '.html', '.css', '.scss', '.sass',
}


def is_risk_file(file_path: str) -> bool:
    """Suffixes not in the whitelist"""
    return os.path.splitext(file_path)[1].lower() not in ALLOWED_EXT


def is_binary_file(file_path: str) -> bool:
    """Unlike file extension detection, it judges from the perspective of file content."""
    with open(file_path, 'rb') as f:
        chunk = f.read(1024)  # Read only the first 1KB
        if b'\0' in chunk:  # Files containing null bytes are typically binary files.
            return True
        return exist_non_text(chunk)


def exist_non_text(content: bytes) -> bool:
    """If the proportion of non-text characters is too high."""
    text_chars = bytearray({7, 8, 9, 10, 12, 13, 27} | set(range(0x20, 0x100)) - {0x7f})
    non_text = sum(1 for b in content if b not in text_chars)
    return non_text / len(content) > 0.3 if content else False
