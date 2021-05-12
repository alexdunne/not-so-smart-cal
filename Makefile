.PHONY:

dev: load_secrets start_docker start_k8s

load_secrets:
	kubectl create secret generic credentials \
	--save-config --dry-run=client \
	--from-env-file .env \
	-o yaml | kubectl apply -f -

start_docker:
	docker-compose up -d

start_k8s:
	skaffold dev --tail