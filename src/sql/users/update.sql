UPDATE users
    SET first_name=$2,
        last_name=$3,
        bio=$4,
        job_title=$5,
        phone=$6,
        username=$7,
        avatar_id=$8,
        website=$9,
        instagram=$10,
        birthday=$11,
        updated_at=NOW()
WHERE id=$1
