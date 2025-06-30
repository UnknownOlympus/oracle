package repository

const GetTaskSummarySQL = `
SELECT
    tt.type_name AS "task_type",
    count(*) AS "count"
FROM
    task_executors te
JOIN
    bot_users bu ON te.executor_id = bu.employee_id
JOIN
    tasks t ON te.task_id = t.task_id
JOIN
    task_types tt ON t.task_type_id = tt.type_id
WHERE
    bu.telegram_id = $1
    AND t.closing_date >= $2
    AND t.closing_date <= $3
GROUP BY
    tt.type_name

UNION ALL

SELECT
    'Total' AS "task_type",
    count(*) AS "count"
FROM
    task_executors te
JOIN
    bot_users bu ON te.executor_id = bu.employee_id
JOIN
    tasks t ON te.task_id = t.task_id
WHERE
    bu.telegram_id = $1
    AND t.closing_date >= $2
    AND t.closing_date <= $3
ORDER BY
    "count" ASC;
`
