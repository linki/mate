go {
    go.run("make build.linux")

    // Push to pierone
    if ("master".equals(env.BRANCH_NAME)) {
        docker.login()
        buildStep("Build and push docker image") {
            stups.run("make build.push")
        }
    }
}
