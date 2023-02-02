module.exports = function(grunt) {
    grunt.initConfig({
        pkg: grunt.file.readJSON('package.json'), // the package file to use

        qunit: { // internal task or name of a plugin (like "qunit")
            all: ['js/tests/*.html']
        },
        watch: {
            files: [
                'js/tests/*.js',
                'js/tests/*.html',
                'tmpl/*.html',
                'js/*.js',
                'src/*.go'
            ],
            tasks: ['qunit', 'shell:buildGo', 'shell:testGo']
        },
        shell: {
            buildGo: {
                command: 'make gobuild',
                options: {stdout: true, stderr: true}
            },
            testGo: {
                command: 'make gotest',
                options: {stdout: true, stderr: true}
            }
        }
    });
    // load up your plugins
    grunt.loadNpmTasks('grunt-contrib-qunit');
    grunt.loadNpmTasks('grunt-contrib-watch');
    grunt.loadNpmTasks('grunt-shell');
    // register one or more task lists (you should ALWAYS have a "default" task list)

    // Disable qunit for now, I can't get it working within docker easily:
    // grunt.registerTask('default', ['qunit']);
    grunt.registerTask('default', []);
};
