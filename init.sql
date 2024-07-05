DROP TABLE IF EXISTS subscribes;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
                       id INT AUTO_INCREMENT PRIMARY KEY,
                       username VARCHAR(200) NOT NULL UNIQUE,
                       firstname VARCHAR(200) NOT NULL,
                       middlename VARCHAR(200) NOT NULL,
                       lastname VARCHAR(200) NOT NULL,
                       password VARCHAR(200) NOT NULL,
                       birthday DATE NOT NULL,
                       telegram VARCHAR(200) NOT NULL UNIQUE,
                       telegramID INT UNIQUE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE subscribes (
                            id INT AUTO_INCREMENT PRIMARY KEY,
                            userID INT NOT NULL,
                            subscriberID INT NOT NULL,
                            FOREIGN KEY (userID) REFERENCES users(id),
                            FOREIGN KEY (subscriberID) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
