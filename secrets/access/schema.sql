CREATE TABLE ACL(
    Fingerprint TEXT    NOT NULL,
    Capability  INT8    NOT NULL,
    Path        TEXT    NOT NULL,
    ValidAfter  INTEGER NOT NULL,
    ValidBefore INTEGER NOT NULL
);

CREATE VIEW ValidACL AS
SELECT Fingerprint, Capability, Path
FROM ACL
WHERE ValidAfter <= unixepoch() AND unixepoch() < ValidBefore
;
