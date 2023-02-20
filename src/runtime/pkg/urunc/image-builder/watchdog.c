#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

int main()
{
    int pid = fork();
    if (pid == 0)
    {
        int err;
        char *argv[3] = {"hello.binary", 0};
        err = execv("./hello.binary", argv); // syscall, libc has simpler wrappers (man exec)
        exit(err);                    // if it got here, it's an error
    }
    else if (pid < 0)
    {
        printf("fork failed with error code %d\n", pid);
        exit(-1);
    }

    int status;
    wait(&status); // simplest one, man wait for others
    printf("child pid was %d, it exited with %d\n", pid, status);
    exit(0);
}

