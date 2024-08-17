update organizations SET
  name=$2,
  description=$3,
  logo_id=$4,
  did=$5
WHERE id=$1