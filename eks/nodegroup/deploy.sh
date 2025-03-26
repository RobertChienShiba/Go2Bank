kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.12.0/deploy/static/provider/aws/deploy.yaml
kubectl apply -f https://raw.githubusercontent.com/external-secrets/external-secrets/main/deploy/crds/bundle.yaml 
kubectl wait --for=condition=available --timeout=180s deployment/external-secrets-webhook -n external-secrets
kubectl wait --for=condition=available --timeout=180s deployment/ingress-nginx-controller -n ingress-nginx
echo "wait for ingress-nginx-controller-admission webhook ready..."
until IP=$(kubectl get endpoints ingress-nginx-controller-admission -n ingress-nginx -o json | jq -r '.subsets[0].addresses[0].ip'); [ $IP != "null" ]; do
  echo "waiting..."
  sleep 5
done
echo "Get IP: $IP, Webhook is ready!!!"
aws eks update-kubeconfig --name go2bank --region ap-northeast-3
kubectl apply -f eks/nodegroup/
kubectl get all --all-namespaces
export ELB_DNS=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o json | jq -r '.status.loadBalancer.ingress[0].hostname')
echo "ELB Domain: $ELB_DNS"
kubectl patch ingress go2bank-ingress-http -n default --type='json'   -p="[{'op': 'replace', 'path': '/spec/rules/0/host', 'value': '${ELB_DNS}'}]"
kubectl get ingress