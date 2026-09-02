-- Rename the permission tree from resources to menus.
--
-- Menu rows still contain both menu and button nodes. This migration changes
-- only identifiers; it preserves every row, UUID, audit field, and grant.
-- It is safe to run after either 001/002 and can be rerun after completion.

-- Rename tables only when the target table is not already present. The
-- dependent foreign keys move with the table and keep their existing data.
DO $$
BEGIN
    IF to_regclass('resources') IS NOT NULL
       AND to_regclass('menus') IS NULL THEN
        ALTER TABLE resources RENAME TO menus;
    END IF;

    IF to_regclass('role_resources') IS NOT NULL
       AND to_regclass('role_menus') IS NULL THEN
        ALTER TABLE role_resources RENAME TO role_menus;
    END IF;
END;
$$;

-- The migration runner currently reapplies all numbered scripts on startup.
-- After this rename, 001 can therefore recreate empty legacy tables before
-- 003 runs again. Remove only those empty duplicates; never discard a row.
DO $$
BEGIN
    IF to_regclass('resources') IS NOT NULL
       AND to_regclass('menus') IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM resources) THEN
            RAISE EXCEPTION
                'both resources and menus contain data; resolve the duplicate tables before renaming';
        END IF;
        IF to_regclass('role_resources') IS NOT NULL
           AND EXISTS (SELECT 1 FROM role_resources) THEN
            RAISE EXCEPTION
                'both role_resources and role_menus contain data; resolve the duplicate tables before renaming';
        END IF;
        IF to_regclass('role_resources') IS NOT NULL THEN
            DROP TABLE role_resources;
        END IF;
        DROP TABLE resources;
    END IF;

    IF to_regclass('role_resources') IS NOT NULL
       AND to_regclass('role_menus') IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM role_resources) THEN
            RAISE EXCEPTION
                'both role_resources and role_menus contain data; resolve the duplicate tables before renaming';
        END IF;
        DROP TABLE role_resources;
    END IF;
END;
$$;

-- Rename columns after the table rename. PostgreSQL updates dependent
-- constraint expressions and foreign-key definitions automatically.
DO $$
BEGIN
    IF to_regclass('menus') IS NOT NULL
       AND EXISTS (
           SELECT 1
             FROM information_schema.columns
            WHERE table_schema = current_schema()
              AND table_name = 'menus'
              AND column_name = 'resource_type'
       )
       AND NOT EXISTS (
           SELECT 1
             FROM information_schema.columns
            WHERE table_schema = current_schema()
              AND table_name = 'menus'
              AND column_name = 'menu_type'
       ) THEN
        ALTER TABLE menus RENAME COLUMN resource_type TO menu_type;
    END IF;

    IF to_regclass('role_menus') IS NOT NULL
       AND EXISTS (
           SELECT 1
             FROM information_schema.columns
            WHERE table_schema = current_schema()
              AND table_name = 'role_menus'
              AND column_name = 'resource_id'
       )
       AND NOT EXISTS (
           SELECT 1
             FROM information_schema.columns
            WHERE table_schema = current_schema()
              AND table_name = 'role_menus'
              AND column_name = 'menu_id'
       ) THEN
        ALTER TABLE role_menus RENAME COLUMN resource_id TO menu_id;
    END IF;
END;
$$;

-- Constraint names are not changed by a table or column rename. Rename them
-- explicitly, while tolerating a partially completed previous execution.
DO $$
DECLARE
    old_name TEXT;
    new_name TEXT;
BEGIN
    IF to_regclass('menus') IS NOT NULL THEN
        FOR old_name, new_name IN
            SELECT * FROM (VALUES
                ('resources_application_not_blank', 'menus_application_not_blank'),
                ('resources_parent_same_application_fk', 'menus_parent_same_application_fk'),
                ('resources_id_application_uq', 'menus_id_application_uq'),
                ('resources_type_check', 'menus_type_check'),
                ('resources_parent_not_self_check', 'menus_parent_not_self_check'),
                ('resources_button_requires_parent_check', 'menus_button_requires_parent_check'),
                ('resources_button_api_path_check', 'menus_button_api_path_check'),
                ('resources_code_not_blank', 'menus_code_not_blank'),
                ('resources_name_not_blank', 'menus_name_not_blank'),
                ('resources_sort_order_check', 'menus_sort_order_check'),
                ('resources_http_method_check', 'menus_http_method_check'),
                ('resources_metadata_object_check', 'menus_metadata_object_check'),
                ('resources_deleted_implies_disabled', 'menus_deleted_implies_disabled')
            ) AS names(old_name, new_name)
        LOOP
            IF EXISTS (
                SELECT 1
                  FROM pg_constraint
                 WHERE conrelid = to_regclass('menus')
                   AND conname = old_name
            )
            AND NOT EXISTS (
                SELECT 1
                  FROM pg_constraint
                 WHERE conrelid = to_regclass('menus')
                   AND conname = new_name
            ) THEN
                EXECUTE format(
                    'ALTER TABLE %I RENAME CONSTRAINT %I TO %I',
                    'menus', old_name, new_name
                );
            END IF;
        END LOOP;
    END IF;

    IF to_regclass('role_menus') IS NOT NULL THEN
        FOR old_name, new_name IN
            SELECT * FROM (VALUES
                ('role_resources_pk', 'role_menus_pk'),
                ('role_resources_role_fk', 'role_menus_role_fk'),
                ('role_resources_resource_fk', 'role_menus_menu_fk')
            ) AS names(old_name, new_name)
        LOOP
            IF EXISTS (
                SELECT 1
                  FROM pg_constraint
                 WHERE conrelid = to_regclass('role_menus')
                   AND conname = old_name
            )
            AND NOT EXISTS (
                SELECT 1
                  FROM pg_constraint
                 WHERE conrelid = to_regclass('role_menus')
                   AND conname = new_name
            ) THEN
                EXECUTE format(
                    'ALTER TABLE %I RENAME CONSTRAINT %I TO %I',
                    'role_menus', old_name, new_name
                );
            END IF;
        END LOOP;
    END IF;
END;
$$;

-- Rename ordinary indexes and the backing indexes of the renamed primary and
-- unique constraints. Constraint-backed index names vary by PostgreSQL
-- version, so missing names are intentionally ignored.
DO $$
DECLARE
    old_name TEXT;
    new_name TEXT;
BEGIN
    FOR old_name, new_name IN
        SELECT * FROM (VALUES
            ('resources_application_code_live_uq', 'menus_application_code_live_uq'),
            ('resources_sibling_name_live_uq', 'menus_sibling_name_live_uq'),
            ('resources_application_tree_idx', 'menus_application_tree_idx'),
            ('resources_enabled_idx', 'menus_enabled_idx'),
            ('resources_id_application_uq', 'menus_id_application_uq'),
            ('role_resources_resource_idx', 'role_menus_menu_idx'),
            ('role_resources_pk', 'role_menus_pk')
        ) AS names(old_name, new_name)
    LOOP
        IF to_regclass(old_name) IS NOT NULL
           AND to_regclass(new_name) IS NULL THEN
            EXECUTE format('ALTER INDEX %I RENAME TO %I', old_name, new_name);
        END IF;
    END LOOP;
END;
$$;

-- Replace the two tree/grant validation functions so their stored PL/pgSQL
-- bodies use the new table and column names. The generic audit function keeps
-- its established name because it is not resource-specific.
DO $$
BEGIN
    IF to_regprocedure('permission_validate_resource_parent()') IS NOT NULL
       AND to_regprocedure('permission_validate_menu_parent()') IS NULL THEN
        ALTER FUNCTION permission_validate_resource_parent()
            RENAME TO permission_validate_menu_parent;
    END IF;

    IF to_regprocedure('permission_validate_role_resource_application()') IS NOT NULL
       AND to_regprocedure('permission_validate_role_menu_application()') IS NULL THEN
        ALTER FUNCTION permission_validate_role_resource_application()
            RENAME TO permission_validate_role_menu_application;
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION permission_validate_menu_parent()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    parent_type VARCHAR(16);
    parent_is_deleted BOOLEAN;
BEGIN
    IF NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT menu_type, is_deleted
      INTO parent_type, parent_is_deleted
      FROM menus
     WHERE id = NEW.parent_id
       AND application = NEW.application;

    IF parent_type IS NULL THEN
        -- The FK will provide the canonical error if this is a new parent.
        RETURN NEW;
    END IF;

    IF parent_type <> 'menu' THEN
        RAISE EXCEPTION 'menu parent % must be a menu', NEW.parent_id
            USING ERRCODE = 'check_violation';
    END IF;

    IF parent_is_deleted THEN
        RAISE EXCEPTION 'menu parent % is deleted', NEW.parent_id
            USING ERRCODE = 'check_violation';
    END IF;

    IF EXISTS (
        WITH RECURSIVE ancestors(id) AS (
            SELECT NEW.parent_id
            UNION ALL
            SELECT m.parent_id
              FROM menus m
              JOIN ancestors a ON a.id = m.id
             WHERE m.application = NEW.application
               AND m.parent_id IS NOT NULL
        )
        SELECT 1 FROM ancestors WHERE id = NEW.id
    ) THEN
        RAISE EXCEPTION 'menu % cannot be its own ancestor', NEW.id
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION permission_validate_role_menu_application()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    role_application VARCHAR(100);
    menu_application VARCHAR(100);
BEGIN
    SELECT application INTO role_application FROM roles WHERE id = NEW.role_id;
    SELECT application INTO menu_application FROM menus WHERE id = NEW.menu_id;
    IF role_application IS DISTINCT FROM menu_application THEN
        RAISE EXCEPTION 'role % and menu % belong to different applications', NEW.role_id, NEW.menu_id
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

-- Recreate triggers under their menu-oriented names. Dropping both possible
-- names first also repairs a partially completed migration without duplicates.
DO $$
BEGIN
    IF to_regclass('menus') IS NOT NULL THEN
        EXECUTE 'DROP TRIGGER IF EXISTS resources_set_updated_at ON menus';
        EXECUTE 'DROP TRIGGER IF EXISTS menus_set_updated_at ON menus';
        EXECUTE 'CREATE TRIGGER menus_set_updated_at
            BEFORE UPDATE ON menus
            FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at()';

        EXECUTE 'DROP TRIGGER IF EXISTS resources_validate_parent ON menus';
        EXECUTE 'DROP TRIGGER IF EXISTS menus_validate_parent ON menus';
        EXECUTE 'CREATE TRIGGER menus_validate_parent
            BEFORE INSERT OR UPDATE OF parent_id, application, menu_type, is_deleted ON menus
            FOR EACH ROW EXECUTE FUNCTION permission_validate_menu_parent()';
    END IF;

    IF to_regclass('role_menus') IS NOT NULL THEN
        EXECUTE 'DROP TRIGGER IF EXISTS role_resources_set_updated_at ON role_menus';
        EXECUTE 'DROP TRIGGER IF EXISTS role_menus_set_updated_at ON role_menus';
        EXECUTE 'CREATE TRIGGER role_menus_set_updated_at
            BEFORE UPDATE ON role_menus
            FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at()';

        EXECUTE 'DROP TRIGGER IF EXISTS role_resources_validate_application ON role_menus';
        EXECUTE 'DROP TRIGGER IF EXISTS role_menus_validate_application ON role_menus';
        EXECUTE 'CREATE TRIGGER role_menus_validate_application
            BEFORE INSERT OR UPDATE OF role_id, menu_id ON role_menus
            FOR EACH ROW EXECUTE FUNCTION permission_validate_role_menu_application()';
    END IF;
END;
$$;

-- If 001/002 were rerun after an earlier 003, their legacy validation
-- functions are now unreferenced because the old triggers were dropped above.
DROP FUNCTION IF EXISTS permission_validate_resource_parent();
DROP FUNCTION IF EXISTS permission_validate_role_resource_application();
