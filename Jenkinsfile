pipeline {
  agent {
    kubernetes {
      label 'efd'
      yamlFile 'podTemplates.yaml'
    }   
  }
  stages {
    stage('Lint') {
      steps {
        container('golangci-lint') {
          sh 'golangci-lint run'
        }
      }
    }
    stage('Build') {
      steps {
        container('golang') {
          sh 'go build -o bin/efd'
        }
      }
    }
  }
}
