resource "google_bigquery_dataset" "tally" {
  dataset_id    = var.dataset_id
  friendly_name = var.friendly_name
  description   = "Tally dataset."
  location      = var.location
}

resource "google_bigquery_table" "scorecard" {
  dataset_id = google_bigquery_dataset.tally.dataset_id
  table_id   = "scorecard"

  deletion_protection = var.scorecard_deletion_protection

  schema = <<EOF
[
 {
  "fields": [
   {
    "mode": "REQUIRED",
    "name": "name",
    "type": "STRING"
   }
  ],
  "mode": "REQUIRED",
  "name": "repo",
  "type": "RECORD"
 },
 {
  "mode": "REQUIRED",
  "name": "score",
  "type": "FLOAT"
 },
 {
  "mode": "REQUIRED",
  "name": "date",
  "type": "DATE"
 }
]
EOF
}
