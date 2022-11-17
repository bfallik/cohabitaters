resource "google_project_service" "people_api" {
  service = "people.googleapis.com"

  disable_dependent_services = true
}
