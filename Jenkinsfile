pipeline {
    agent any
    environment {
        GH_TOKEN  = credentials('GITHUB_CREDENTIALS_ID')
        GOOGLE_APPLICATION_CREDENTIALS = credentials('webapp-operator-gcp')
        PROJECT_ID = 'csye7125-cloud-003'
        CLUSTER_NAME = 'csye7125-cloud-003-gke'
        REGION = 'us-west1'
        KUBE_DEPLOYMENT_NAME = 'webappcr-controller-manager'
        OP_NAMESPACE = 'webappcr-system'
        WEBAPP_NAMESPACE = 'webapp'
    }
    stages {
        stage('Fetch GitHub Credentials') {
            steps {
                script {
                    // Define credentials for GitHub
                    withCredentials([usernamePassword(credentialsId: 'GITHUB_CREDENTIALS_ID', usernameVariable: 'githubUsername', passwordVariable: 'githubToken')]) {
                        git branch: 'main', credentialsId: 'GITHUB_CREDENTIALS_ID', url: 'https://github.com/cyse7125-fall2023-group2/webapp-operator'        
                    }
                }
            }
        }
    
       stage('build operator images'){
        steps{
            script{
                withCredentials([usernamePassword(credentialsId: 'DOCKER_HUB_ID', usernameVariable: 'dockerHubUsername', passwordVariable: 'dockerHubPassword')]) {
                  sh """
                    docker login -u ${dockerHubUsername} -p ${dockerHubPassword}
                    make docker-build docker-push IMG=sumanthksai/webapp-operator:latest
                     """
                }

            }
        }
       }
       
    stage('make deploy'){
        steps{
            script{
                    withCredentials([file(credentialsId: 'webapp-operator-gcp', variable: 'SA_KEY')]) {

                  sh """
                    gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}
                    gcloud config set project ${PROJECT_ID}
                    gcloud container clusters get-credentials ${CLUSTER_NAME} --region ${REGION} --project ${PROJECT_ID}                    
                    """

                                                                // Check if the deployment exists
                    def deploymentExists = sh(script: "kubectl get deployment ${KUBE_DEPLOYMENT_NAME} --namespace=${OP_NAMESPACE}", returnStatus: true)

                    // If the deployment exists, delete it
                    if (deploymentExists == 0) {
                        sh "kubectl delete deployment ${KUBE_DEPLOYMENT_NAME} --namespace=${OP_NAMESPACE}"
                    }

                    sh "kubectl delete all --selector app=kafka-producer -n ${WEBAPP_NAMESPACE} -o name"

                    sh "make deploy IMG=sumanthksai/webapp-operator:latest"
                    sh "kubectl label namespace ${OP_NAMESPACE} istio-injection=enabled"
                }

            }
        }
       }
    }
}