pipeline {
    agent any
    environment {
        GH_TOKEN  = credentials('GITHUB_CREDENTIALS_ID')
        GOOGLE_APPLICATION_CREDENTIALS = credentials('webapp-operator')
        PROJECT_ID = 'csye7125-cloud-002'
        CLUSTER_NAME = 'csye7125-cloud-002-gke'
        REGION = 'us-east1'

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
                    make docker-build docker-push IMG=sumanthksai/webapp-operator:latest
                     """
                }

            }
        }
       }
       
    stage('make deploy'){
        steps{
            script{
                    withCredentials([file(credentialsId: 'webapp-operator', variable: 'SA_KEY')]) {

                  sh """
                    gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}
                    gcloud config set project ${PROJECT_ID}
                    gcloud container clusters get-credentials ${CLUSTER_NAME} --region ${REGION} --project ${PROJECT_ID}
                    make deploy IMG=sumanthksai/webapp-operator:latest
                    """
                }

            }
        }
       }
    }
}