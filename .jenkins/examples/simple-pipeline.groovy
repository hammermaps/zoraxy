// Simple example Jenkinsfile for Zoraxy
// This is a minimal configuration for quick setup

pipeline {
    agent any
    
    environment {
        PROJECT_NAME = 'zoraxy'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Build') {
            steps {
                dir('src') {
                    sh '''
                        go version
                        go mod tidy
                        go build -o zoraxy
                    '''
                }
            }
        }
        
        stage('Archive') {
            steps {
                archiveArtifacts artifacts: 'src/zoraxy', fingerprint: true
            }
        }
    }
    
    post {
        success {
            echo 'Build completed successfully!'
        }
        failure {
            echo 'Build failed!'
        }
    }
}
