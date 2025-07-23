CREATE TABLE subs (
    sub_id UUID PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL,
    price INTEGER NOT NULL,
    user_id UUID NOT NULL,
    start_date DATE NOT NULL, 
    end_date DATE 
);

CREATE FUNCTION sum_in_period(
    IN filter_start DATE,
    IN filter_end DATE,
    IN filter_user_id UUID DEFAULT NULL,
    IN filter_service_name VARCHAR(255) DEFAULT NULL
)
RETURNS INTEGER AS $$
DECLARE
    sum INTEGER := 0;
BEGIN
    WITH filtered AS (
        SELECT
            sub_id,
            GREATEST(start_date, filter_start) AS overlap_start,
            LEAST(COALESCE(end_date, filter_end), filter_end) AS overlap_end,
            price
        FROM subs
        WHERE
            (filter_user_id IS NULL OR user_id = filter_user_id)
            AND (filter_service_name IS NULL OR service_name = filter_service_name)
            AND start_date <= filter_end
            AND (end_date IS NULL OR end_date >= filter_start)
    )
    SELECT SUM(
        ((DATE_PART('year', overlap_end) - DATE_PART('year', overlap_start)) * 12 +
        (DATE_PART('month', overlap_end) - DATE_PART('month', overlap_start)) + 1
        ) * price
    )
    INTO sum
    FROM filtered
    WHERE overlap_start <= overlap_end;

    RETURN COALESCE(sum, 0);
END;
$$ LANGUAGE plpgsql;