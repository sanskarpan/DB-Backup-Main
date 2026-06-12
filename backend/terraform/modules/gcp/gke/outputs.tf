output "cluster_id" {
  description = "ID of the GKE cluster"
  value       = google_container_cluster.main.id
}

output "cluster_name" {
  description = "Name of the GKE cluster"
  value       = google_container_cluster.main.name
}

output "cluster_endpoint" {
  description = "Endpoint of the GKE cluster"
  value       = google_container_cluster.main.endpoint
  sensitive   = true
}

output "cluster_ca_certificate" {
  description = "CA certificate of the GKE cluster"
  value       = google_container_cluster.main.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

output "system_node_pool_name" {
  description = "Name of the system node pool"
  value       = google_container_node_pool.system.name
}

output "backup_node_pool_name" {
  description = "Name of the backup node pool"
  value       = var.create_backup_node_pool ? google_container_node_pool.backup[0].name : null
}
