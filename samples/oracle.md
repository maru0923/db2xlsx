---
database:
  type: oracle
  host: oracle.example.com
  port: 1521
  service_name: ORCLPDB1
  username: report_user
  password: ${DB_PASSWORD}
  connect_timeout_seconds: 30
---

# Oracle report queries

```sql name="Employee List"
SELECT
  EMPLOYEE_ID,
  EMPLOYEE_NAME,
  DEPARTMENT_ID,
  HIRE_DATE,
  SALARY
FROM
  EMPLOYEES
WHERE
  STATUS = 'ACTIVE'
ORDER BY
  EMPLOYEE_ID
```

```sql name="Department Counts"
SELECT
  DEPARTMENT_ID,
  COUNT(*) AS EMPLOYEE_COUNT
FROM
  EMPLOYEES
WHERE
  STATUS = 'ACTIVE'
GROUP BY
  DEPARTMENT_ID
ORDER BY
  DEPARTMENT_ID
```
