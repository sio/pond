CREATE VIEW IF NOT EXISTS allowed AS
SELECT
    key.key           AS key,
    access.access     AS access,
    MAX(allow_get)    AS allow_get,
    MAX(allow_set)    AS allow_set,
    MAX(allow_delete) AS allow_delete
FROM
    key

    LEFT JOIN user
    ON key.user = user.user

    LEFT JOIN usergroup
    ON key.user = usergroup.user

    LEFT JOIN access
    ON usergroup.usergroup == access.usergroup

    WHERE user.admin = 0 AND user.disabled = 0

    GROUP BY
        key.key,
        access.access
;
