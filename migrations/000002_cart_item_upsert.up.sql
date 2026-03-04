-- Deduplicate existing cart_items by merging quantities for rows that share
-- the same (cart_id, product_id, variant_id) before adding the unique index.
WITH dupes AS (
    SELECT MIN(id::text)::uuid AS keep_id, SUM(quantity) AS total_qty, cart_id, product_id, variant_id
    FROM cart_items
    GROUP BY cart_id, product_id, variant_id
    HAVING COUNT(*) > 1
)
UPDATE cart_items ci
SET quantity = d.total_qty
FROM dupes d
WHERE ci.id = d.keep_id;

DELETE FROM cart_items ci
WHERE EXISTS (
    SELECT 1
    FROM cart_items ci2
    WHERE ci2.cart_id    = ci.cart_id
      AND ci2.product_id = ci.product_id
      AND ci2.variant_id IS NOT DISTINCT FROM ci.variant_id
      AND ci2.id::text < ci.id::text
);

-- Unique index to support upsert (quantity accumulation) when the same
-- product/variant is added to a cart more than once.
-- NULL variant_id values are coerced to the nil UUID so that the expression
-- is deterministic and can serve as a conflict target.
CREATE UNIQUE INDEX uq_cart_items_cart_product_variant
    ON cart_items (cart_id, product_id, (COALESCE(variant_id, '00000000-0000-0000-0000-000000000000'::uuid)));
