UPDATE orders o
SET customer_id = c.id
FROM customers c
WHERE o.customer_id IS NULL
  AND o.billing_address->>'email' IS NOT NULL
  AND lower(o.billing_address->>'email') = lower(c.email);
