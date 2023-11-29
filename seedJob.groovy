pipelineJob('webapp-helm-chart-pipeline') {
    description('My Pipeline Job Description')
    definition {
        cpsScm {
            scriptPath('Jenkinsfile') // Reference the Jenkinsfile in your SCM
            scm {
                git {
                    remote {
                        url('https://github.com/cyse7125-fall2023-group2/webapp-operator.git')
                        credentials('WEBHOOK_CREDENTIAL') // Specify your GitHub credentials ID
                    }
                    branch('main') // Specify the branch you want to build
                }
            }
        }
    }
}