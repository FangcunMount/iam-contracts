-- Receipt is audit and rollback evidence. Preserve it when rolling back schema
-- version; deleting it would make an applied role migration unrecoverable.
SELECT 1;
