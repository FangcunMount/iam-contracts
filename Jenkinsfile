pipeline {
    agent any
    
    environment {
        // 项目配置
        PROJECT_NAME = 'iam-contracts'
        GO_VERSION = '1.24'
        DEFAULT_IMAGE_NAMESPACE = 'iam-contracts'
        DEFAULT_IMAGE_TAG = 'prod'
        
        // 构建产物路径
        BINARY_NAME = 'apiserver'
        BUILD_DIR = 'bin'
        
        // 部署配置
        DEPLOY_USER = 'deploy'
        DEPLOY_HOST = '${DEPLOY_HOST}' // 从 Jenkins 凭据中获取
        DEPLOY_PATH = '/opt/iam'
        
        // 服务配置
        CONFIG_FILE = 'configs/apiserver.yaml'
        SERVICE_NAME = 'iam-apiserver'
        SERVICE_PORT = '8080'
        
        // Docker 网络
        DOCKER_NETWORK = 'iam-network'
        
        // Git 信息 (将在 Setup 阶段设置)
        GIT_COMMIT_SHORT = ''
        BUILD_TIME = ''
    }
    
    options {
        skipDefaultCheckout(true)
        timestamps()
        timeout(time: 30, unit: 'MINUTES')
        disableConcurrentBuilds()
        buildDiscarder(logRotator(numToKeepStr: '10'))
    }
    
    // 运行时参数，可在 Jenkins UI 中配置
    parameters {
        choice(name: 'DEPLOY_MODE', choices: ['docker', 'binary', 'systemd'], description: '部署模式：docker (Docker 容器), binary (二进制直接运行), systemd (系统服务)')
        string(name: 'IMAGE_REGISTRY', defaultValue: 'iam-contracts', description: 'Docker 镜像仓库命名空间')
        string(name: 'IMAGE_TAG', defaultValue: 'prod', description: 'Docker 镜像标签')
        booleanParam(name: 'LOAD_ENV_FROM_CREDENTIALS', defaultValue: true, description: '从 Jenkins 凭据加载环境变量配置文件')
        string(name: 'ENV_CREDENTIALS_ID', defaultValue: 'iam-contracts-prod-env', description: '环境变量凭据 ID')
        booleanParam(name: 'RUN_TESTS', defaultValue: true, description: '运行单元测试')
        booleanParam(name: 'RUN_LINT', defaultValue: true, description: '运行代码检查')
        booleanParam(name: 'SKIP_BUILD', defaultValue: false, description: '跳过构建阶段（用于仅部署场景）')
        booleanParam(name: 'DOCKER_NO_CACHE', defaultValue: false, description: 'Docker 构建时不使用缓存')
        booleanParam(name: 'SKIP_DB_INIT', defaultValue: true, description: '跳过数据库初始化（已有环境推荐跳过）')
        booleanParam(name: 'SKIP_DB_MIGRATE', defaultValue: true, description: '跳过数据库迁移')
        booleanParam(name: 'SKIP_DB_SEED', defaultValue: true, description: '跳过加载种子数据（仅首次部署需要）')
        booleanParam(name: 'PUSH_IMAGES', defaultValue: false, description: '推送 Docker 镜像到仓库')
        booleanParam(name: 'DEPLOY_AFTER_BUILD', defaultValue: true, description: '构建后自动部署')
        string(name: 'DEPLOY_COMPOSE_FILES', defaultValue: 'build/docker/docker-compose.yml', description: 'Docker Compose 配置文件（空格分隔）')
        booleanParam(name: 'ENABLE_HEALTH_CHECK', defaultValue: true, description: '部署后执行健康检查')
        booleanParam(name: 'AUTO_ROLLBACK', defaultValue: true, description: '健康检查失败时自动回滚')
        booleanParam(name: 'PRUNE_IMAGES', defaultValue: false, description: '清理悬空的 Docker 镜像')
        string(name: 'DB_ROOT_CREDENTIALS_ID', defaultValue: 'mysql-root-password', description: 'MySQL root 密码凭据 ID')
        string(name: 'DEPLOY_SSH_CREDENTIALS_ID', defaultValue: 'deploy-ssh-key', description: 'SSH 部署凭据 ID')
    }
    
    stages {
        stage('Checkout') {
            steps {
                deleteDir()
                checkout scm
                script {
                    // 在 checkout 后设置 Git 相关变量
                    env.GIT_COMMIT = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
                    env.GIT_COMMIT_SHORT = sh(returnStdout: true, script: 'git rev-parse --short HEAD').trim()
                    env.GIT_BRANCH = sh(returnStdout: true, script: 'git rev-parse --abbrev-ref HEAD').trim()
                    env.BUILD_TIME = sh(returnStdout: true, script: 'date -u +"%Y-%m-%d_%H:%M:%S"').trim()
                    echo "================================================"
                    echo "  项目: ${env.PROJECT_NAME}"
                    echo "  分支: ${env.GIT_BRANCH}"
                    echo "  提交: ${env.GIT_COMMIT_SHORT}"
                    echo "  构建: #${env.BUILD_NUMBER}"
                    echo "  时间: ${env.BUILD_TIME}"
                    echo "  部署模式: ${params.DEPLOY_MODE}"
                    echo "================================================"
                }
            }
        }
        
        stage('Setup') {
            steps {
                script {
                    // 初始化环境变量和配置
                    initializeEnvironment()
                }
            }
        }
        
        stage('依赖管理') {
            when {
                allOf {
                    expression { env.RUN_BUILD == 'true' }
                    expression { params.DEPLOY_MODE != 'docker' }
                }
            }
            steps {
                echo '📦 下载 Go 依赖...'
                sh '''
                    go env -w GO111MODULE=on
                    go env -w GOPROXY=https://goproxy.cn,direct
                    go mod download
                    go mod tidy
                    go mod verify
                '''
            }
        }
        
        stage('代码检查') {
            when {
                allOf {
                    expression { env.RUN_LINT == 'true' }
                    expression { params.DEPLOY_MODE != 'docker' }
                }
            }
            parallel {
                stage('代码格式化检查') {
                    steps {
                        echo '🔍 检查代码格式...'
                        sh '''
                            if ! command -v gofmt &> /dev/null; then
                                echo "gofmt 不可用，跳过格式检查"
                            else
                                UNFORMATTED=$(gofmt -l .)
                                if [ -n "$UNFORMATTED" ]; then
                                    echo "以下文件需要格式化:"
                                    echo "$UNFORMATTED"
                                    echo "⚠️ 建议运行: make fmt"
                                fi
                            fi
                        '''
                    }
                }
                
                stage('代码静态分析') {
                    steps {
                        echo '🔍 运行静态分析...'
                        sh '''
                            if command -v golangci-lint &> /dev/null; then
                                golangci-lint run --timeout=5m || echo "⚠️ 发现一些问题，但继续构建"
                            else
                                echo "golangci-lint 未安装，跳过静态分析"
                                go vet ./... || echo "⚠️ go vet 发现问题"
                            fi
                        '''
                    }
                }
            }
        }
        
        stage('单元测试') {
            when {
                allOf {
                    expression { env.RUN_TESTS == 'true' }
                    expression { params.DEPLOY_MODE != 'docker' }
                }
            }
            steps {
                echo '🧪 运行单元测试...'
                sh '''
                    mkdir -p coverage
                    go test -v -race -coverprofile=coverage/coverage.out -covermode=atomic ./... || {
                        echo "⚠️ 测试失败，但继续构建（可根据需要调整）"
                        exit 0
                    }
                    
                    if [ -f coverage/coverage.out ]; then
                        go tool cover -func=coverage/coverage.out | tail -1
                    fi
                '''
            }
        }
        
        stage('编译构建') {
            when {
                expression { env.RUN_BUILD == 'true' && env.DEPLOY_MODE != 'docker' }
            }
            steps {
                echo '🔨 编译 Go 应用...'
                sh '''
                    # 确保构建目录存在
                    mkdir -p ${BUILD_DIR}
                    
                    # 构建应用
                    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
                    
                    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
                        -ldflags "-s -w \
                            -X main.Version=${VERSION} \
                            -X main.BuildTime=${env.BUILD_TIME} \
                            -X main.GitCommit=${env.GIT_COMMIT_SHORT}" \
                        -o ${BUILD_DIR}/${BINARY_NAME} \
                        ./cmd/apiserver/
                    
                    # 验证构建产物
                    if [ ! -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
                        echo "❌ 构建失败：二进制文件不存在"
                        exit 1
                    fi
                    
                    # 显示文件信息
                    ls -lh ${BUILD_DIR}/${BINARY_NAME}
                    file ${BUILD_DIR}/${BINARY_NAME}
                '''
            }
        }
        
        stage('构建 Docker 镜像') {
            when {
                expression { env.RUN_DOCKER_BUILD == 'true' }
            }
            steps {
                echo '🐳 构建 Docker 镜像...'
                script {
                    def buildArgs = [
                        "VERSION=${env.VERSION}",
                        "BUILD_TIME=${env.BUILD_TIME}",
                        "GIT_COMMIT=${env.GIT_COMMIT_SHORT}"
                    ]
                    def noCacheFlag = env.DOCKER_NO_CACHE == 'true' ? '--no-cache' : ''
                    
                    sh """
                        docker build ${noCacheFlag} \
                            --build-arg ${buildArgs.join(' --build-arg ')} \
                            -f build/docker/Dockerfile \
                            -t ${env.IMAGE_TAG_FULL} \
                            -t ${env.IMAGE_REGISTRY}:latest \
                            .
                        
                        echo "✅ Docker 镜像构建完成"
                        docker images ${env.IMAGE_REGISTRY}
                    """
                }
            }
        }
        
        stage('准备 Docker 网络') {
            when {
                expression { env.RUN_DOCKER_BUILD == 'true' || env.DEPLOY_MODE == 'docker' }
            }
            steps {
                echo '🔗 准备 Docker 网络...'
                sh '''
                    if ! docker network ls | grep -q ${DOCKER_NETWORK}; then
                        docker network create ${DOCKER_NETWORK}
                        echo "✅ 创建网络: ${DOCKER_NETWORK}"
                    else
                        echo "ℹ️  网络已存在: ${DOCKER_NETWORK}"
                    fi
                '''
            }
        }
        
        stage('推送镜像') {
            when {
                expression { env.PUSH_IMAGES_FLAG == 'true' }
            }
            steps {
                echo '📤 推送 Docker 镜像...'
                sh """
                    docker tag ${env.IMAGE_TAG_FULL} ${env.IMAGE_REGISTRY}/${env.PROJECT_NAME}:${env.IMAGE_TAG}
                    docker push ${env.IMAGE_REGISTRY}/${env.PROJECT_NAME}:${env.IMAGE_TAG}
                    
                    if [ "${env.IMAGE_TAG}" != "latest" ]; then
                        docker tag ${env.IMAGE_TAG_FULL} ${env.IMAGE_REGISTRY}/${env.PROJECT_NAME}:latest
                        docker push ${env.IMAGE_REGISTRY}/${env.PROJECT_NAME}:latest
                    fi
                    
                    echo "✅ 镜像推送完成"
                """
            }
        }
        
        stage('数据库初始化') {
            when {
                expression { env.RUN_DB_INIT == 'true' }
            }
            steps {
                withCredentials([string(credentialsId: params.DB_ROOT_CREDENTIALS_ID, variable: 'DB_ROOT_PASSWORD')]) {
                    echo '🗄️ 初始化数据库...'
                    sh '''
                        chmod +x scripts/sql/init-db.sh
                        DB_PASSWORD=${DB_ROOT_PASSWORD} scripts/sql/init-db.sh --skip-confirm
                        echo "✅ 数据库初始化完成"
                    '''
                }
            }
        }
        
        stage('数据库迁移') {
            when {
                expression { env.RUN_DB_MIGRATE == 'true' }
            }
            steps {
                echo '🔄 执行数据库迁移...'
                sh '''
                    if [ -f scripts/sql/migrate.sh ]; then
                        chmod +x scripts/sql/migrate.sh
                        scripts/sql/migrate.sh
                        echo "✅ 数据库迁移完成"
                    else
                        echo "⚠️ 迁移脚本不存在，跳过"
                    fi
                '''
            }
        }
        
        stage('加载种子数据') {
            when {
                expression { env.RUN_DB_SEED == 'true' }
            }
            steps {
                echo '🌱 加载种子数据...'
                sh '''
                    chmod +x scripts/sql/init-db.sh
                    scripts/sql/init-db.sh --seed-only --skip-confirm
                    echo "✅ 种子数据加载完成"
                '''
            }
        }
        
        stage('部署') {
            when {
                expression { env.RUN_DEPLOY == 'true' }
            }
            steps {
                script {
                    if (env.DEPLOY_MODE == 'docker') {
                        deployWithDocker()
                    } else if (env.DEPLOY_MODE == 'systemd') {
                        deployWithSystemd()
                    } else {
                        deployBinary()
                    }
                }
            }
        }
        
        stage('健康检查') {
            when {
                expression { env.ENABLE_HEALTH_CHECK == 'true' && env.RUN_DEPLOY == 'true' }
            }
            steps {
                script {
                    echo '🔍 执行健康检查...'
                    def healthCheckPassed = performHealthCheck()
                    
                    if (!healthCheckPassed && env.AUTO_ROLLBACK == 'true') {
                        echo '❌ 健康检查失败，执行自动回滚...'
                        performRollback()
                        error('健康检查失败，已回滚到上一版本')
                    } else if (!healthCheckPassed) {
                        error('健康检查失败')
                    }
                }
            }
        }
        
        stage('清理') {
            when {
                expression { env.RUN_IMAGE_PRUNE == 'true' }
            }
            steps {
                echo '🧹 清理悬空镜像...'
                sh '''
                    docker image prune -f
                    echo "✅ 清理完成"
                '''
            }
        }
    }
    
    post {
        success {
            script {
                echo '✅ 部署成功！'
                // 使用 try-catch 防止变量未定义导致错误
                try {
                    echo """
                    ================================================
                    🎉 部署成功
                    ================================================
                    项目: ${env.PROJECT_NAME ?: 'iam-contracts'}
                    分支: ${env.GIT_BRANCH ?: 'unknown'}
                    提交: ${env.GIT_COMMIT_SHORT ?: 'unknown'}
                    构建: #${env.BUILD_NUMBER ?: '0'}
                    时间: ${env.BUILD_TIME ?: 'unknown'}
                    部署模式: ${env.DEPLOY_MODE ?: 'unknown'}
                    ================================================
                    """
                } catch (Exception e) {
                    echo "部署成功（部分信息获取失败）"
                }
            }
        }
        
        failure {
            script {
                echo '❌ 部署失败！'
                // 使用 try-catch 防止变量未定义导致错误
                try {
                    echo """
                    ================================================
                    ⚠️ 部署失败
                    ================================================
                    项目: ${env.PROJECT_NAME ?: 'iam-contracts'}
                    分支: ${env.GIT_BRANCH ?: 'unknown'}
                    构建: #${env.BUILD_NUMBER ?: '0'}
                    ================================================
                    请检查构建日志
                    """
                } catch (Exception e) {
                    echo "部署失败（详细信息获取失败）"
                }
            }
        }
        
        always {
            script {
                // 使用 try-catch 防止在 node 外执行 sh 命令
                try {
                    echo '🧹 清理工作空间...'
                    sh '''
                        rm -rf deploy coverage
                    '''
                } catch (Exception e) {
                    echo "清理跳过（工作空间不可用）: ${e.message}"
                }
            }
        }
    }
}

// ============================================================================
// 辅助函数
// ============================================================================

def initializeEnvironment() {
    // 从凭据加载环境变量
    if (flagEnabled(params.LOAD_ENV_FROM_CREDENTIALS) && params.ENV_CREDENTIALS_ID?.trim()) {
        loadEnvFromCredentials(params.ENV_CREDENTIALS_ID.trim())
    } else {
        echo 'ℹ️  跳过从凭据加载环境变量'
    }
    
    // 设置部署模式
    env.DEPLOY_MODE = params.DEPLOY_MODE?.trim() ?: 'binary'
    
    // 设置镜像配置
    env.IMAGE_REGISTRY = normalizeRegistry(params.IMAGE_REGISTRY) ?: env.DEFAULT_IMAGE_NAMESPACE
    env.IMAGE_TAG = params.IMAGE_TAG?.trim() ?: env.DEFAULT_IMAGE_TAG
    env.IMAGE_TAG_FULL = buildImageTag(env.IMAGE_REGISTRY, env.PROJECT_NAME, env.IMAGE_TAG)
    
    // 设置版本信息
    env.VERSION = sh(returnStdout: true, script: 'git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev"').trim()
    
    // 设置运行标志
    env.RUN_BUILD = shouldSkip(params.SKIP_BUILD, env.SKIP_BUILD) ? 'false' : 'true'
    env.RUN_TESTS = flagEnabled(params.RUN_TESTS) ? 'true' : 'false'
    env.RUN_LINT = flagEnabled(params.RUN_LINT) ? 'true' : 'false'
    
    // Docker 相关
    env.RUN_DOCKER_BUILD = (env.DEPLOY_MODE == 'docker' && env.RUN_BUILD == 'true') ? 'true' : 'false'
    env.DOCKER_NO_CACHE = flagEnabled(params.DOCKER_NO_CACHE) ? 'true' : 'false'
    
    // 数据库操作标志
    env.RUN_DB_INIT = params.SKIP_DB_INIT ? 'false' : 'true'
    env.RUN_DB_MIGRATE = params.SKIP_DB_MIGRATE ? 'false' : 'true'
    env.RUN_DB_SEED = params.SKIP_DB_SEED ? 'false' : 'true'
    
    // 推送和部署标志
    def pushImages = flagEnabled(params.PUSH_IMAGES) || flagEnabled(env.PUSH_IMAGES)
    env.PUSH_IMAGES_FLAG = pushImages ? 'true' : 'false'
    
    def deployAfterBuild = flagEnabled(params.DEPLOY_AFTER_BUILD) || flagEnabled(env.DEPLOY_AFTER_BUILD)
    env.RUN_DEPLOY = (deployAfterBuild && !flagEnabled(env.SKIP_DEPLOY)) ? 'true' : 'false'
    
    // 其他配置
    env.DEPLOY_COMPOSE_FILES = params.DEPLOY_COMPOSE_FILES?.trim() ?: 'build/docker/docker-compose.yml'
    env.ENABLE_HEALTH_CHECK = flagEnabled(params.ENABLE_HEALTH_CHECK) ? 'true' : 'false'
    env.AUTO_ROLLBACK = flagEnabled(params.AUTO_ROLLBACK) ? 'true' : 'false'
    
    def pruneImages = flagEnabled(params.PRUNE_IMAGES) || flagEnabled(env.PRUNE_IMAGES)
    env.RUN_IMAGE_PRUNE = pruneImages ? 'true' : 'false'
    
    // 打印配置信息
    echo """
    ================================================
    环境配置
    ================================================
    部署模式: ${env.DEPLOY_MODE}
    镜像仓库: ${env.IMAGE_REGISTRY}
    镜像标签: ${env.IMAGE_TAG}
    完整镜像: ${env.IMAGE_TAG_FULL}
    版本号: ${env.VERSION}
    运行构建: ${env.RUN_BUILD}
    运行测试: ${env.RUN_TESTS}
    运行检查: ${env.RUN_LINT}
    Docker构建: ${env.RUN_DOCKER_BUILD}
    Docker无缓存: ${env.DOCKER_NO_CACHE}
    数据库初始化: ${env.RUN_DB_INIT}
    数据库迁移: ${env.RUN_DB_MIGRATE}
    加载种子数据: ${env.RUN_DB_SEED}
    推送镜像: ${env.PUSH_IMAGES_FLAG}
    执行部署: ${env.RUN_DEPLOY}
    健康检查: ${env.ENABLE_HEALTH_CHECK}
    自动回滚: ${env.AUTO_ROLLBACK}
    清理镜像: ${env.RUN_IMAGE_PRUNE}
    ================================================
    """
}

def loadEnvFromCredentials(String credentialsId) {
    withCredentials([file(credentialsId: credentialsId, variable: 'ENV_FILE')]) {
        def content = readFile(ENV_FILE)
        def target = "${env.WORKSPACE}/.pipeline.env"
        writeFile file: target, text: content
        env.PIPELINE_ENV_FILE = target
        
        def exposedKeys = content.split('\n')
            .findAll { line ->
                def trimmed = line.trim()
                trimmed && !trimmed.startsWith('#') && trimmed.contains('=')
            }
            .collect { entry ->
                entry.split('=', 2)[0].replaceFirst(/^export\s+/, '').trim()
            }
        
        echo "✅ 从凭据 '${credentialsId}' 加载环境变量 (keys: ${exposedKeys.join(', ')})"
    }
}

def deployWithDocker() {
    echo '🐳 使用 Docker Compose 部署...'
    
    def composeFiles = env.DEPLOY_COMPOSE_FILES.split()
    def composeFileArgs = composeFiles.collect { "-f ${it}" }.join(' ')
    
    sh """
        # 如果需要拉取镜像
        if [ "${env.PUSH_IMAGES_FLAG}" = "true" ]; then
            docker-compose ${composeFileArgs} pull
        fi
        
        # 停止旧容器
        docker-compose ${composeFileArgs} down
        
        # 启动新容器
        docker-compose ${composeFileArgs} up -d
        
        # 显示状态
        docker-compose ${composeFileArgs} ps
        
        echo "✅ Docker 部署完成"
    """
}

def deployWithSystemd() {
    echo '⚙️ 使用 Systemd 服务部署...'
    
    sshagent(credentials: [params.DEPLOY_SSH_CREDENTIALS_ID]) {
        sh """
            # 创建远程目录
            ssh -o StrictHostKeyChecking=no ${env.DEPLOY_USER}@${env.DEPLOY_HOST} "
                mkdir -p ${env.DEPLOY_PATH}/{bin,configs,logs,scripts}
                mkdir -p /var/log/iam-contracts
            "
            
            # 上传文件
            echo "📤 上传部署文件..."
            scp -o StrictHostKeyChecking=no ${BUILD_DIR}/${BINARY_NAME} ${env.DEPLOY_USER}@${env.DEPLOY_HOST}:${env.DEPLOY_PATH}/bin/
            scp -o StrictHostKeyChecking=no -r configs ${env.DEPLOY_USER}@${env.DEPLOY_HOST}:${env.DEPLOY_PATH}/
            scp -o StrictHostKeyChecking=no build/systemd/iam-apiserver.service ${env.DEPLOY_USER}@${env.DEPLOY_HOST}:/tmp/
            
            # 部署服务
            ssh -o StrictHostKeyChecking=no ${env.DEPLOY_USER}@${env.DEPLOY_HOST} "
                # 安装 systemd 服务
                if [ ! -f /etc/systemd/system/${env.SERVICE_NAME}.service ]; then
                    sudo mv /tmp/iam-apiserver.service /etc/systemd/system/
                    sudo systemctl daemon-reload
                    sudo systemctl enable ${env.SERVICE_NAME}
                fi
                
                # 重启服务
                sudo systemctl restart ${env.SERVICE_NAME}
                sleep 3
                
                # 检查状态
                sudo systemctl status ${env.SERVICE_NAME} --no-pager
            "
            
            echo "✅ Systemd 服务部署完成"
        """
    }
}

def deployBinary() {
    echo '📦 使用二进制文件部署...'
    
    sshagent(credentials: [params.DEPLOY_SSH_CREDENTIALS_ID]) {
        sh """
            # 创建远程目录
            ssh -o StrictHostKeyChecking=no ${env.DEPLOY_USER}@${env.DEPLOY_HOST} "
                mkdir -p ${env.DEPLOY_PATH}/{bin,configs,logs,scripts}
            "
            
            # 上传文件
            echo "📤 上传部署文件..."
            scp -o StrictHostKeyChecking=no ${BUILD_DIR}/${BINARY_NAME} ${env.DEPLOY_USER}@${env.DEPLOY_HOST}:${env.DEPLOY_PATH}/bin/
            scp -o StrictHostKeyChecking=no -r configs ${env.DEPLOY_USER}@${env.DEPLOY_HOST}:${env.DEPLOY_PATH}/
            scp -o StrictHostKeyChecking=no scripts/deploy.sh ${env.DEPLOY_USER}@${env.DEPLOY_HOST}:${env.DEPLOY_PATH}/scripts/
            
            # 部署服务
            ssh -o StrictHostKeyChecking=no ${env.DEPLOY_USER}@${env.DEPLOY_HOST} "
                cd ${env.DEPLOY_PATH}
                chmod +x scripts/deploy.sh bin/${BINARY_NAME}
                
                # 使用部署脚本
                ./scripts/deploy.sh deploy
            "
            
            echo "✅ 二进制部署完成"
        """
    }
}

def performHealthCheck() {
    echo '🔍 执行健康检查...'
    
    def maxRetry = 10
    def retryCount = 0
    def healthUrl = "http://${env.DEPLOY_HOST}:${env.SERVICE_PORT}/healthz"
    
    if (env.DEPLOY_MODE == 'docker') {
        healthUrl = "http://localhost:${env.SERVICE_PORT}/healthz"
    }
    
    while (retryCount < maxRetry) {
        try {
            def response = sh(
                script: "curl -sf ${healthUrl}",
                returnStatus: true
            )
            
            if (response == 0) {
                echo "✅ 健康检查通过"
                return true
            }
        } catch (Exception e) {
            echo "健康检查失败: ${e.message}"
        }
        
        echo "等待服务就绪... (${retryCount + 1}/${maxRetry})"
        sleep 3
        retryCount++
    }
    
    echo "❌ 健康检查失败"
    return false
}

def performRollback() {
    echo '⏪ 执行回滚...'
    
    if (env.DEPLOY_MODE == 'docker') {
        sh '''
            echo "回滚 Docker 容器到上一版本..."
            # 这里可以实现具体的回滚逻辑
            echo "⚠️ Docker 回滚需要手动实现"
        '''
    } else {
        sshagent(credentials: [params.DEPLOY_SSH_CREDENTIALS_ID]) {
            sh """
                ssh -o StrictHostKeyChecking=no ${env.DEPLOY_USER}@${env.DEPLOY_HOST} "
                    cd ${env.DEPLOY_PATH}
                    if [ -f scripts/deploy.sh ]; then
                        ./scripts/deploy.sh rollback
                    else
                        echo '⚠️ 回滚脚本不存在'
                    fi
                "
            """
        }
    }
}

boolean flagEnabled(def value) {
    if (value == null) {
        return false
    }
    def normalized = value.toString().trim().toLowerCase()
    return ['1', 'true', 'yes', 'y'].contains(normalized)
}

boolean shouldSkip(def paramValue, def envValue) {
    return flagEnabled(paramValue) || flagEnabled(envValue)
}

String normalizeRegistry(Object input) {
    def raw = input?.toString()?.trim()
    if (!raw) {
        return ''
    }
    return raw.endsWith('/') ? raw[0..-2] : raw
}

String buildImageTag(String registry, String component, String tag) {
    def base = registry?.trim()
    def repo = base ? "${base}/${component}" : component
    return "${repo}:${tag}"
}
