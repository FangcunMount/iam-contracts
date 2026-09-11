-- The retirement ledger is permanent audit evidence. Authorization rollback
-- is performed by condition-authz-retire; it must not drop this ledger.
SELECT iam_condition_retirement_requires_coordinated_restore();
