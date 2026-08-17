const { notify } = require('./notify');
const {
    getEnvs,
    getEnv,
    addEnvs,
    addEnv,
    updateEnv,
    deleteEnvs,
    deleteEnv
} = require('./env');
const {
    getTasks,
    getTask,
    updateTask,
    deleteTask,
    executeTask,
    stopTask,
    getLastResults
} = require('./task');

module.exports = {
    notify,
    getEnvs,
    getEnv,
    addEnvs,
    addEnv,
    updateEnv,
    deleteEnvs,
    deleteEnv,
    getTasks,
    getTask,
    updateTask,
    deleteTask,
    executeTask,
    stopTask,
    getLastResults
};
