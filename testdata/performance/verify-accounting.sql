SELECT c.id,c.committed_spend_minor,
       COALESCE(sum(b.amount_minor) FILTER (WHERE b.outcome='APPROVED'),0)::bigint AS ledger_spend
FROM campaigns c
LEFT JOIN budget_consumption_commands b ON b.campaign_id=c.id
GROUP BY c.id
HAVING c.committed_spend_minor < 0
    OR c.committed_spend_minor > c.budget_amount_minor
    OR c.committed_spend_minor <>
       COALESCE(sum(b.amount_minor) FILTER (WHERE b.outcome='APPROVED'),0)::bigint;
