;; thanks stack overflow
;; https://stackoverflow.com/questions/4012321/how-can-i-access-the-path-to-the-current-directory-in-an-emacs-directory-variabl
((nil . ((eval . (set (make-local-variable 'my-project-path)
                      (file-name-directory
                       (let ((d (dir-locals-find-file ".")))
                         (if (stringp d) d (car d))))))
         (eval . (setenv "GOPATH" my-project-path))
         (eval . (message "Project directory set to `%s'." my-project-path)))))
