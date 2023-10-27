
# AWS EKS Kubernetes Network

This is a network suggestion for running aws eks kubernetes. It is low cost since it only runs one nat gateway. It is designed to be able to practice with eks running on top




## Diagram

![Alt text](/diagram/network.png "quickDiagram")




## Deployment

```bash
curl -fsSL https://get.pulumi.com | sh 

export AWS_ACCESS_KEY_ID=<YOUR_ACCESS_KEY_ID> && \
export AWS_SECRET_ACCESS_KEY=<YOUR_SECRET_ACCESS_KEY> && \ 
export AWS_REGION=<YOUR_AWS_REGION>

pulumi up 