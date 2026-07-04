---
database:
  type: postgresql
  host: postgres.example.com
  port: 5432
  database: salesdb
  schema: public
  username: report_user
  password: ${DB_PASSWORD}
  sslmode: require
  connect_timeout_seconds: 30
---

# PostgreSQL report queries

```sql name="Orders"
SELECT
  order_id,
  customer_id,
  order_date,
  total_amount,
  status
FROM
  orders
WHERE
  order_date >= DATE '2026-01-01'
ORDER BY
  order_date,
  order_id
```

```sql name="Sales By Customer"
SELECT
  customer_id,
  COUNT(*) AS order_count,
  SUM(total_amount) AS total_sales,
  MAX(order_date) AS last_order_date
FROM
  orders
WHERE
  order_date >= DATE '2026-01-01'
GROUP BY
  customer_id
ORDER BY
  total_sales DESC
```
