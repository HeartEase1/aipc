-- Allow multiple proxies to share one backup.
-- The previous unique constraint on backup_proxy_id made backups exclusive
-- and caused Ent to clear incoming references when a shared backup was edited.
DO $$
DECLARE
    rec record;
BEGIN
    FOR rec IN
        SELECT c.conname
        FROM pg_constraint c
        WHERE c.conrelid = 'proxies'::regclass
          AND c.contype = 'u'
          AND pg_get_constraintdef(c.oid) ILIKE '%backup_proxy_id%'
    LOOP
        EXECUTE format('ALTER TABLE proxies DROP CONSTRAINT IF EXISTS %I', rec.conname);
    END LOOP;

    FOR rec IN
        SELECT DISTINCT i.relname AS index_name
        FROM pg_index x
        JOIN pg_class t ON t.oid = x.indrelid
        JOIN pg_class i ON i.oid = x.indexrelid
        JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY (x.indkey)
        WHERE t.relname = 'proxies'
          AND x.indisunique
          AND NOT x.indisprimary
          AND a.attname = 'backup_proxy_id'
    LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I', rec.index_name);
    END LOOP;
END $$;
