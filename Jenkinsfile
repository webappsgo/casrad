pipeline {
    agent none

    triggers {
        // Daily build at 3am UTC (matches GitHub Actions daily.yml)
        cron('0 3 * * *')
    }

    environment {
        PROJECTNAME = 'casrad'
        PROJECTORG  = 'casapps'
        BINDIR      = 'binaries'
        RELDIR      = 'releases'
        GODIR       = "/tmp/${PROJECTORG}/go"
        GOCACHE     = "/tmp/${PROJECTORG}/go/build"

        // =========================================================================
        // GIT PROVIDER CONFIGURATION
        // Uncomment ONE block below based on your git hosting platform
        // =========================================================================

        // ----- GITHUB (default) -----
        GIT_FQDN  = 'github.com'
        GIT_TOKEN = credentials('github-token')
        REGISTRY  = "ghcr.io/${PROJECTORG}/${PROJECTNAME}"

        // ----- GITEA / FORGEJO (self-hosted) -----
        // GIT_FQDN  = 'git.example.com'
        // GIT_TOKEN = credentials('gitea-token')
        // REGISTRY  = "${GIT_FQDN}/${PROJECTORG}/${PROJECTNAME}"

        // ----- GITLAB (gitlab.com or self-hosted) -----
        // GIT_FQDN  = 'gitlab.com'
        // GIT_TOKEN = credentials('gitlab-token')
        // REGISTRY  = "registry.${GIT_FQDN}/${PROJECTORG}/${PROJECTNAME}"

        // ----- DOCKER HUB -----
        // GIT_FQDN  = 'github.com'
        // GIT_TOKEN = credentials('github-token')
        // REGISTRY  = "docker.io/${PROJECTORG}/${PROJECTNAME}"

        // =========================================================================
    }

    stages {
        stage('Checkout') {
            agent { label 'amd64' }
            steps {
                checkout scm
            }
        }

        stage('Setup') {
            agent { label 'amd64' }
            steps {
                script {
                    if (env.TAG_NAME) {
                        env.BUILD_TYPE = 'release'
                        env.VERSION    = env.TAG_NAME.replaceFirst('^v', '')
                    } else if (env.BRANCH_NAME == 'beta') {
                        env.BUILD_TYPE = 'beta'
                        env.VERSION    = sh(script: 'date -u +"%Y%m%d%H%M%S"', returnStdout: true).trim() + '-beta'
                    } else if (env.BRANCH_NAME == 'main' || env.BRANCH_NAME == 'master') {
                        env.BUILD_TYPE = 'daily'
                        env.VERSION    = sh(script: 'date -u +"%Y%m%d%H%M%S"', returnStdout: true).trim()
                    } else {
                        env.BUILD_TYPE = 'dev'
                        env.VERSION    = sh(script: 'date -u +"%Y%m%d%H%M%S"', returnStdout: true).trim() + '-dev'
                    }
                    env.COMMIT_ID    = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                    env.BUILD_DATE   = sh(script: 'date +"%a %b %d, %Y at %H:%M:%S %Z"', returnStdout: true).trim()
                    // OFFICIALSITE: site.txt wins; otherwise use Jenkins credential or leave empty
                    env.OFFICIALSITE = sh(script: '[ -f site.txt ] && cat site.txt || echo "${OFFICIALSITE:-}"', returnStdout: true).trim()
                    env.LDFLAGS      = "-s -w -X 'main.Version=${env.VERSION}' -X 'main.CommitID=${env.COMMIT_ID}' -X 'main.BuildDate=${env.BUILD_DATE}' -X 'main.OfficialSite=${env.OFFICIALSITE}'"
                    env.HAS_CLI      = sh(script: '[ -d src/client ] && echo true || echo false', returnStdout: true).trim()
                    env.HAS_AGENT    = sh(script: '[ -d src/agent ]  && echo true || echo false', returnStdout: true).trim()
                }
                sh 'mkdir -p "${BINDIR}" "${RELDIR}"'
                echo "Build type: ${BUILD_TYPE}, Version: ${VERSION}, Commit: ${COMMIT_ID}"
            }
        }

        stage('Build') {
            parallel {
                stage('Linux AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=linux \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-linux-amd64" ./src
                        '''
                    }
                }
                stage('Linux ARM64') {
                    agent { label 'arm64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=linux \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-linux-arm64" ./src
                        '''
                    }
                }
                stage('Darwin AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=darwin \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-darwin-amd64" ./src
                        '''
                    }
                }
                stage('Darwin ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=darwin \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-darwin-arm64" ./src
                        '''
                    }
                }
                stage('Windows AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=windows \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-windows-amd64.exe" ./src
                        '''
                    }
                }
                stage('Windows ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=windows \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-windows-arm64.exe" ./src
                        '''
                    }
                }
                stage('FreeBSD AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=freebsd \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-freebsd-amd64" ./src
                        '''
                    }
                }
                stage('FreeBSD ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=freebsd \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-freebsd-arm64" ./src
                        '''
                    }
                }
            }
        }

        stage('Build CLI') {
            when {
                expression { env.HAS_CLI == 'true' }
            }
            parallel {
                stage('CLI Linux AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=linux \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-cli-linux-amd64" ./src/client
                        '''
                    }
                }
                stage('CLI Linux ARM64') {
                    agent { label 'arm64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=linux \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-cli-linux-arm64" ./src/client
                        '''
                    }
                }
                stage('CLI Darwin AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=darwin \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-cli-darwin-amd64" ./src/client
                        '''
                    }
                }
                stage('CLI Darwin ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=darwin \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-cli-darwin-arm64" ./src/client
                        '''
                    }
                }
                stage('CLI Windows AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=windows \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-cli-windows-amd64.exe" ./src/client
                        '''
                    }
                }
                stage('CLI Windows ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=windows \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-cli-windows-arm64.exe" ./src/client
                        '''
                    }
                }
                stage('CLI FreeBSD AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=freebsd \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-cli-freebsd-amd64" ./src/client
                        '''
                    }
                }
                stage('CLI FreeBSD ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=freebsd \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-cli-freebsd-arm64" ./src/client
                        '''
                    }
                }
            }
        }

        stage('Build Agent') {
            when {
                expression { env.HAS_AGENT == 'true' }
            }
            parallel {
                stage('Agent Linux AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=linux \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-agent-linux-amd64" ./src/agent
                        '''
                    }
                }
                stage('Agent Linux ARM64') {
                    agent { label 'arm64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=linux \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-agent-linux-arm64" ./src/agent
                        '''
                    }
                }
                stage('Agent Darwin AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=darwin \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-agent-darwin-amd64" ./src/agent
                        '''
                    }
                }
                stage('Agent Darwin ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=darwin \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-agent-darwin-arm64" ./src/agent
                        '''
                    }
                }
                stage('Agent Windows AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=windows \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-agent-windows-amd64.exe" ./src/agent
                        '''
                    }
                }
                stage('Agent Windows ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=windows \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-agent-windows-arm64.exe" ./src/agent
                        '''
                    }
                }
                stage('Agent FreeBSD AMD64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=freebsd \
                                -e GOARCH=amd64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-agent-freebsd-amd64" ./src/agent
                        '''
                    }
                }
                stage('Agent FreeBSD ARM64') {
                    agent { label 'amd64' }
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                -e GOOS=freebsd \
                                -e GOARCH=arm64 \
                                golang:alpine \
                                go build -ldflags "${LDFLAGS}" -o "${BINDIR}/${PROJECTNAME}-agent-freebsd-arm64" ./src/agent
                        '''
                    }
                }
            }
        }

        stage('Test') {
            agent { label 'amd64' }
            steps {
                sh '''
                    docker run --rm \
                        --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                        -v "${WORKSPACE}:/build" \
                        -v "${GOCACHE}:/root/.cache/go-build" \
                        -v "${GODIR}:/go" \
                        -w /build \
                        -e CGO_ENABLED=0 \
                        golang:alpine \
                        go test -v -cover ./...
                '''
            }
        }

        stage('Security') {
            agent { label 'amd64' }
            parallel {
                stage('govulncheck') {
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/build" \
                                -v "${GOCACHE}:/root/.cache/go-build" \
                                -v "${GODIR}:/go" \
                                -w /build \
                                -e CGO_ENABLED=0 \
                                golang:alpine \
                                sh -c "go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./..."
                        '''
                    }
                }
                stage('Secret Scan') {
                    steps {
                        sh '''
                            docker run --rm \
                                --name "${PROJECTNAME}-$(tr -dc 'a-z0-9' </dev/urandom | head -c8)" \
                                -v "${WORKSPACE}:/workdir" \
                                -w /workdir \
                                trufflesecurity/trufflehog:latest \
                                filesystem --directory=. --fail
                        '''
                    }
                }
            }
        }

        stage('Docker') {
            agent { label 'amd64' }
            steps {
                script {
                    def tags = "-t ${REGISTRY}:${COMMIT_ID}"

                    if (env.BUILD_TYPE == 'release') {
                        def yymm = new Date().format('yyMM')
                        tags += " -t ${REGISTRY}:${VERSION}"
                        tags += " -t ${REGISTRY}:latest"
                        tags += " -t ${REGISTRY}:${yymm}"
                    } else if (env.BUILD_TYPE == 'beta') {
                        tags += " -t ${REGISTRY}:beta"
                        tags += " -t ${REGISTRY}:devel"
                    } else {
                        tags += " -t ${REGISTRY}:devel"
                    }

                    sh """
                        echo "\${GIT_TOKEN}" | docker login ${REGISTRY.split('/')[0]} -u ${PROJECTORG} --password-stdin
                    """

                    sh """
                        docker buildx create --name ${PROJECTNAME}-builder --use 2>/dev/null || docker buildx use ${PROJECTNAME}-builder
                        docker buildx build \
                            -f docker/Dockerfile \
                            --platform linux/amd64,linux/arm64 \
                            --build-arg VERSION="${VERSION}" \
                            --build-arg COMMIT_ID="${COMMIT_ID}" \
                            --build-arg BUILD_DATE="${BUILD_DATE}" \
                            --label "org.opencontainers.image.vendor=${PROJECTORG}" \
                            --label "org.opencontainers.image.authors=${PROJECTORG}" \
                            --label "org.opencontainers.image.title=${PROJECTNAME}" \
                            --label "org.opencontainers.image.base.name=${PROJECTNAME}" \
                            --label "org.opencontainers.image.description=${PROJECTNAME} - standard image (alpine)" \
                            --label "org.opencontainers.image.licenses=MIT" \
                            --label "org.opencontainers.image.version=${VERSION}" \
                            --label "org.opencontainers.image.created=${BUILD_DATE}" \
                            --label "org.opencontainers.image.revision=${COMMIT_ID}" \
                            --label "org.opencontainers.image.url=https://${GIT_FQDN}/${PROJECTORG}/${PROJECTNAME}" \
                            --label "org.opencontainers.image.source=https://${GIT_FQDN}/${PROJECTORG}/${PROJECTNAME}" \
                            --label "org.opencontainers.image.documentation=https://${GIT_FQDN}/${PROJECTORG}/${PROJECTNAME}" \
                            --annotation "manifest:org.opencontainers.image.vendor=${PROJECTORG}" \
                            --annotation "manifest:org.opencontainers.image.authors=${PROJECTORG}" \
                            --annotation "manifest:org.opencontainers.image.title=${PROJECTNAME}" \
                            --annotation "manifest:org.opencontainers.image.base.name=${PROJECTNAME}" \
                            --annotation "manifest:org.opencontainers.image.description=${PROJECTNAME} - standard image (alpine)" \
                            --annotation "manifest:org.opencontainers.image.licenses=MIT" \
                            --annotation "manifest:org.opencontainers.image.version=${VERSION}" \
                            --annotation "manifest:org.opencontainers.image.created=${BUILD_DATE}" \
                            --annotation "manifest:org.opencontainers.image.revision=${COMMIT_ID}" \
                            --annotation "manifest:org.opencontainers.image.url=https://${GIT_FQDN}/${PROJECTORG}/${PROJECTNAME}" \
                            --annotation "manifest:org.opencontainers.image.source=https://${GIT_FQDN}/${PROJECTORG}/${PROJECTNAME}" \
                            --annotation "manifest:org.opencontainers.image.documentation=https://${GIT_FQDN}/${PROJECTORG}/${PROJECTNAME}" \
                            ${tags} \
                            --push \
                            .
                    """
                }
            }
        }

        stage('Release: Stable') {
            agent { label 'amd64' }
            when {
                expression { env.BUILD_TYPE == 'release' }
            }
            steps {
                sh '''
                    echo "${VERSION}" > "${RELDIR}/version.txt"
                    for f in "${BINDIR}/${PROJECTNAME}-"*; do
                        [ -f "$f" ] || continue
                        cp "$f" "${RELDIR}/"
                    done
                    tar --exclude='.git' --exclude='.github' --exclude='.gitea' \
                        --exclude='.forgejo' --exclude='binaries' --exclude='releases' \
                        --exclude='*.tar.gz' \
                        -czf "${RELDIR}/${PROJECTNAME}-${VERSION}-source.tar.gz" .
                '''
                archiveArtifacts artifacts: 'releases/*', fingerprint: true
            }
        }

        stage('Release: Beta') {
            agent { label 'amd64' }
            when {
                expression { env.BUILD_TYPE == 'beta' }
            }
            steps {
                sh '''
                    echo "${VERSION}" > "${RELDIR}/version.txt"
                    for f in "${BINDIR}/${PROJECTNAME}-"*; do
                        [ -f "$f" ] || continue
                        cp "$f" "${RELDIR}/"
                    done
                '''
                archiveArtifacts artifacts: 'releases/*', fingerprint: true
            }
        }

        stage('Release: Daily') {
            agent { label 'amd64' }
            when {
                expression { env.BUILD_TYPE == 'daily' }
            }
            steps {
                sh '''
                    echo "${VERSION}" > "${RELDIR}/version.txt"
                    for f in "${BINDIR}/${PROJECTNAME}-"*; do
                        [ -f "$f" ] || continue
                        cp "$f" "${RELDIR}/"
                    done
                '''
                archiveArtifacts artifacts: 'releases/*', fingerprint: true
            }
        }
    }

    post {
        always {
            cleanWs()
        }
    }
}
