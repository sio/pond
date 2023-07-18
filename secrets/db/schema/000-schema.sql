-- Secrets storage
CREATE TABLE IF NOT EXISTS secret(
    -- Namespace and key uniquely identify a secret
    namespace TEXT NOT NULL,
    key TEXT NOT NULL,

    -- Encrypted value
    value BLOB,

    -- Name of access control list
    access string,

    -- Expiration date. If not provided, will be set to current timestamp
    -- making the value automatically expired
    expires INTEGER NOT NULL DEFAULT (unixepoch()),

    -- Last modification time
    modified INTEGER NOT NULL DEFAULT (unixepoch()),

    -- Constraints
    PRIMARY KEY (namespace, key)
);


-- Automatic updates for modification time field
CREATE TRIGGER IF NOT EXISTS secret_mtime AFTER UPDATE ON secret BEGIN
    UPDATE secret SET
        modified = unixepoch()
    WHERE namespace = new.namespace AND key = new.key;
END;


-- Namespaces provide default values for access control and for expiration delay
CREATE TABLE IF NOT EXISTS namespace(
    namespace TEXT NOT NULL PRIMARY KEY,
    default_access TEXT,
    default_maxage INTEGER DEFAULT (60*60*24*365)
);


-- Access control lists
CREATE TABLE IF NOT EXISTS access(
    -- ACL name
    access TEXT NOT NULL,

    -- Reference to user group
    usergroup TEXT NOT NULL,

    -- Permissions bitmask
    permissions INTEGER,

    -- Constraints
    PRIMARY KEY (access, usergroup)
);


-- User accounts
CREATE TABLE IF NOT EXISTS usergroup(
    usergroup TEXT NOT NULL,
    user TEXT NOT NULL,
    PRIMARY KEY (usergroup, user)
);
CREATE TABLE IF NOT EXISTS user(
    user TEXT NOT NULL PRIMARY KEY,
    admin BOOLEAN DEFAULT FALSE,
    disabled BOOLEAN DEFAULT FALSE
);
CREATE TABLE IF NOT EXISTS key(
    key TEXT PRIMARY KEY,
    user TEXT NOT NULL
);


-- Database schema migrations
CREATE TABLE IF NOT EXISTS migration(
    schema TEXT NOT NULL,
    timestamp DATETIME DEFAULT (datetime('now', 'utc'))
);
