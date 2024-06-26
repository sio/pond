CREATE TABLE IF NOT EXISTS ACL(
    Fingerprint TEXT    NOT NULL,
    Capability  INT8    NOT NULL,
    Path        TEXT    NOT NULL,
    Priority    INT16   NOT NULL,
    ValidAfter  INTEGER NOT NULL,
    ValidBefore INTEGER NOT NULL
);

CREATE VIEW IF NOT EXISTS ValidACL AS
SELECT Fingerprint, Capability, Path, Priority, ValidAfter
FROM ACL
WHERE ValidAfter <= unixepoch() AND unixepoch() < ValidBefore
;
