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
                command: 'go build -o build/rtfblog src/*.go',
                options: {
                    stdout: true,
                    stderr: true
                }
            },
            testGo: {
                command: 'go test ./src/...',
                options: {
                    stdout: true,
                    stderr: true
                }
            }
        }
    });
    // load up your plugins
    grunt.loadNpmTasks('grunt-contrib-qunit');
    grunt.loadNpmTasks('grunt-contrib-watch');
    grunt.loadNpmTasks('grunt-shell');
    // register one or more task lists (you should ALWAYS have a "default" task list)
    grunt.registerTask('default', ['qunit', 'shell:buildGo', 'shell:testGo']);
};
