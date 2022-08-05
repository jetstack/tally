variable "dataset_id" {
  type        = string
  description = "Dataset ID"
  default     = "tally"
}

variable "friendly_name" {
  type        = string
  description = "Dataset friendly name"
  default     = "tally"
}

variable "location" {
  type        = string
  description = "Dataset location"
  default     = "US"
}

variable "scorecard_deletion_protection" {
  type        = bool
  description = "Deletion protection for the scorecard table"
  default     = true
}
