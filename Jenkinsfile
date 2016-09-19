go {
    go.run("make build.linux")

    // Push to pierone
    if ("feature/adapters-vendor".equals(env.BRANCH_NAME)) {
        docker.login()
        buildStep("Build and push docker image") {
            go.run("make build.push")
        }
    }
}
