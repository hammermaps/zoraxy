pipeline {
    agent any
    
    parameters {
        choice(
            name: 'BUILD_TYPE',
            choices: ['release', 'development'],
            description: 'Build type: release or development'
        )
        booleanParam(
            name: 'BUILD_DOCKER',
            defaultValue: true,
            description: 'Build Docker images'
        )
        booleanParam(
            name: 'RUN_TESTS',
            defaultValue: true,
            description: 'Run tests before building'
        )
    }
    
    environment {
        GO_VERSION = '1.24'
        PROJECT_NAME = 'zoraxy'
        BUILD_DIR = 'src'
        DIST_DIR = 'src/dist'
    }
    
    stages {
        stage('Checkout') {
            steps {
                echo 'Checking out source code...'
                checkout scm
                script {
                    env.GIT_COMMIT_SHORT = sh(
                        script: "git rev-parse --short HEAD",
                        returnStdout: true
                    ).trim()
                    env.BUILD_TIMESTAMP = sh(
                        script: "date '+%Y%m%d-%H%M%S'",
                        returnStdout: true
                    ).trim()
                }
            }
        }
        
        stage('Setup Go') {
            steps {
                echo "Setting up Go ${GO_VERSION}..."
                sh '''
                    go version
                    cd ${BUILD_DIR}
                    go mod download
                    go mod tidy
                '''
            }
        }
        
        stage('Run Tests') {
            when {
                expression { params.RUN_TESTS == true }
            }
            steps {
                echo 'Running tests...'
                sh '''
                    cd ${BUILD_DIR}
                    go test -v ./...
                '''
            }
        }
        
        stage('Build Binaries') {
            parallel {
                stage('Build Linux AMD64') {
                    steps {
                        echo 'Building Linux AMD64...'
                        sh '''
                            cd ${BUILD_DIR}
                            GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/zoraxy_linux_amd64 -ldflags "-s -w" -trimpath
                        '''
                    }
                }
                
                stage('Build Linux ARM64') {
                    steps {
                        echo 'Building Linux ARM64...'
                        sh '''
                            cd ${BUILD_DIR}
                            GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/zoraxy_linux_arm64 -ldflags "-s -w" -trimpath
                        '''
                    }
                }
                
                stage('Build Linux ARM') {
                    steps {
                        echo 'Building Linux ARM (ARMv6)...'
                        sh '''
                            cd ${BUILD_DIR}
                            GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o dist/zoraxy_linux_arm -ldflags "-s -w" -trimpath
                        '''
                    }
                }
                
                stage('Build Windows AMD64') {
                    steps {
                        echo 'Building Windows AMD64...'
                        sh '''
                            cd ${BUILD_DIR}
                            GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/zoraxy_windows_amd64.exe -ldflags "-s -w" -trimpath
                        '''
                    }
                }
                
                stage('Build macOS AMD64') {
                    steps {
                        echo 'Building macOS AMD64...'
                        sh '''
                            cd ${BUILD_DIR}
                            GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o dist/zoraxy_darwin_amd64 -ldflags "-s -w" -trimpath
                        '''
                    }
                }
                
                stage('Build FreeBSD AMD64') {
                    steps {
                        echo 'Building FreeBSD AMD64...'
                        sh '''
                            cd ${BUILD_DIR}
                            GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -o dist/zoraxy_freebsd_amd64 -ldflags "-s -w" -trimpath
                        '''
                    }
                }
            }
        }
        
        stage('Build Docker Image') {
            when {
                expression { params.BUILD_DOCKER == true }
            }
            steps {
                echo 'Building Docker image...'
                script {
                    sh '''
                        # Copy source to docker directory
                        cp -r src/ docker/src/
                        
                        # Build Docker image
                        cd docker
                        docker build -t zoraxy:${BUILD_TIMESTAMP} -t zoraxy:latest .
                        
                        # Clean up
                        rm -rf src/
                    '''
                }
            }
        }
        
        stage('Generate Checksums') {
            steps {
                echo 'Generating checksums...'
                sh '''
                    cd ${DIST_DIR}
                    sha256sum zoraxy_* > zoraxy_checksums.sha256
                '''
            }
        }
        
        stage('Archive Artifacts') {
            steps {
                echo 'Archiving build artifacts...'
                archiveArtifacts artifacts: 'src/dist/**/*', fingerprint: true
            }
        }
    }
    
    post {
        success {
            echo 'Build completed successfully!'
            emailext (
                subject: "SUCCESS: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]'",
                body: """
                    Build Status: SUCCESS
                    Job: ${env.JOB_NAME}
                    Build Number: ${env.BUILD_NUMBER}
                    Build URL: ${env.BUILD_URL}
                    Git Commit: ${env.GIT_COMMIT_SHORT}
                """,
                to: '${DEFAULT_RECIPIENTS}',
                attachLog: false
            )
        }
        failure {
            echo 'Build failed!'
            emailext (
                subject: "FAILURE: Job '${env.JOB_NAME} [${env.BUILD_NUMBER}]'",
                body: """
                    Build Status: FAILURE
                    Job: ${env.JOB_NAME}
                    Build Number: ${env.BUILD_NUMBER}
                    Build URL: ${env.BUILD_URL}
                    Git Commit: ${env.GIT_COMMIT_SHORT}
                """,
                to: '${DEFAULT_RECIPIENTS}',
                attachLog: true
            )
        }
        always {
            echo 'Cleaning up workspace...'
            cleanWs()
        }
    }
}
